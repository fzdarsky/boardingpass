/**
 * TLS Override for React Native's HTTP Request Handler
 *
 * Injects both session-level and task-level authentication challenge handlers
 * into RCTHTTPRequestHandler to implement Trust-On-First-Use (TOFU) certificate
 * pinning for self-signed certificates.
 *
 * TLS server trust is a session-level challenge, so NSURLSession dispatches it
 * to URLSession:didReceiveChallenge:completionHandler: (no task: parameter).
 * We inject both levels for complete coverage.
 *
 * RCTHTTPRequestHandler conforms to NSURLSessionDataDelegate but does NOT
 * implement either authentication challenge handler, so we add them directly
 * via class_addMethod (not swizzling — there's nothing to swap with).
 *
 * Behavior:
 *   - Pin exists AND fingerprint matches → ACCEPT (trusted connection)
 *   - Pin exists AND fingerprint differs → REJECT (certificate changed)
 *   - No pin exists → ACCEPT (TOFU: first connection, JS layer handles trust dialog)
 *
 * The pin store is shared with CertificatePinningModule.swift via UserDefaults
 * under the key "com.boardingpass.certificatePins".
 */

#import <Foundation/Foundation.h>
#import <React/RCTHTTPRequestHandler.h>
#import <objc/runtime.h>
#import <CommonCrypto/CommonDigest.h>
#import <Security/Security.h>

static NSString *const kPinStoreKey = @"com.boardingpass.certificatePins";

/**
 * Shared challenge handling logic used by both session-level and task-level handlers.
 */
static void BPHandleChallenge(NSURLAuthenticationChallenge *challenge,
                              void (^completionHandler)(NSURLSessionAuthChallengeDisposition, NSURLCredential *_Nullable))
{
  NSLog(@"[BoardingPass TLS] Challenge received: %@ for %@:%ld",
        challenge.protectionSpace.authenticationMethod,
        challenge.protectionSpace.host,
        (long)challenge.protectionSpace.port);

  // Only intercept server trust challenges (TLS certificate validation)
  if (![challenge.protectionSpace.authenticationMethod
        isEqualToString:NSURLAuthenticationMethodServerTrust]) {
    completionHandler(NSURLSessionAuthChallengePerformDefaultHandling, nil);
    return;
  }

  SecTrustRef serverTrust = challenge.protectionSpace.serverTrust;
  if (serverTrust == NULL) {
    completionHandler(NSURLSessionAuthChallengeCancelAuthenticationChallenge, nil);
    return;
  }

  // Extract the leaf certificate from the chain
  SecCertificateRef certificate = NULL;
  if (@available(iOS 15.0, *)) {
    CFArrayRef certChain = SecTrustCopyCertificateChain(serverTrust);
    if (certChain == NULL || CFArrayGetCount(certChain) == 0) {
      if (certChain) CFRelease(certChain);
      completionHandler(NSURLSessionAuthChallengeCancelAuthenticationChallenge, nil);
      return;
    }
    certificate = (SecCertificateRef)CFRetain(CFArrayGetValueAtIndex(certChain, 0));
    CFRelease(certChain);
  } else {
    // Fallback for iOS < 15 (deprecated API)
    if (SecTrustGetCertificateCount(serverTrust) == 0) {
      completionHandler(NSURLSessionAuthChallengeCancelAuthenticationChallenge, nil);
      return;
    }
    certificate = (SecCertificateRef)CFRetain(SecTrustGetCertificateAtIndex(serverTrust, 0));
  }

  NSData *certData = (__bridge_transfer NSData *)SecCertificateCopyData(certificate);
  CFRelease(certificate);

  // Compute SHA-256 fingerprint of the certificate
  uint8_t digest[CC_SHA256_DIGEST_LENGTH];
  CC_SHA256(certData.bytes, (CC_LONG)certData.length, digest);

  NSMutableString *fingerprint = [NSMutableString stringWithCapacity:CC_SHA256_DIGEST_LENGTH * 2];
  for (int i = 0; i < CC_SHA256_DIGEST_LENGTH; i++) {
    [fingerprint appendFormat:@"%02x", digest[i]];
  }

  // Look up the pin store
  NSDictionary *pins = [[NSUserDefaults standardUserDefaults] objectForKey:kPinStoreKey];
  NSString *host = challenge.protectionSpace.host;
  NSInteger port = challenge.protectionSpace.port;

  // Try host:port first, then host-only
  NSString *hostPortKey = [NSString stringWithFormat:@"%@:%ld", host, (long)port];
  NSString *pinnedFingerprint = pins[hostPortKey] ?: pins[host];

  if (pinnedFingerprint != nil) {
    // Pin exists — validate fingerprint
    if ([fingerprint isEqualToString:[pinnedFingerprint lowercaseString]]) {
      NSLog(@"[BoardingPass TLS] Pin match for %@:%ld", host, (long)port);
      NSURLCredential *credential = [NSURLCredential credentialForTrust:serverTrust];
      completionHandler(NSURLSessionAuthChallengeUseCredential, credential);
    } else {
      NSLog(@"[BoardingPass TLS] Pin MISMATCH for %@:%ld — rejecting", host, (long)port);
      completionHandler(NSURLSessionAuthChallengeCancelAuthenticationChallenge, nil);
    }
  } else {
    // No pin — TOFU: accept on first connection.
    NSLog(@"[BoardingPass TLS] TOFU: accepting unpinned cert for %@:%ld (fingerprint: %@)",
          host, (long)port, fingerprint);
    NSURLCredential *credential = [NSURLCredential credentialForTrust:serverTrust];
    completionHandler(NSURLSessionAuthChallengeUseCredential, credential);
  }
}

@implementation RCTHTTPRequestHandler (CertificatePinning)

+ (void)load
{
  static dispatch_once_t onceToken;
  dispatch_once(&onceToken, ^{
    Class cls = objc_getClass("RCTHTTPRequestHandler");
    if (!cls) return;

    // Inject session-level challenge handler (for TLS server trust)
    SEL sessionChallengeSel = @selector(URLSession:didReceiveChallenge:completionHandler:);
    SEL sessionImplSel = @selector(bp_URLSession:didReceiveChallenge:completionHandler:);
    Method sessionImpl = class_getInstanceMethod(cls, sessionImplSel);
    if (sessionImpl) {
      BOOL added = class_addMethod(cls, sessionChallengeSel,
                                   method_getImplementation(sessionImpl),
                                   method_getTypeEncoding(sessionImpl));
      if (added) {
        NSLog(@"[BoardingPass TLS] Injected session-level challenge handler");
      } else {
        NSLog(@"[BoardingPass TLS] Session-level handler exists, swizzling");
        Method original = class_getInstanceMethod(cls, sessionChallengeSel);
        if (original) method_exchangeImplementations(original, sessionImpl);
      }
    }

    // Also inject task-level challenge handler (fallback)
    SEL taskChallengeSel = @selector(URLSession:task:didReceiveChallenge:completionHandler:);
    SEL taskImplSel = @selector(bp_URLSession:task:didReceiveChallenge:completionHandler:);
    Method taskImpl = class_getInstanceMethod(cls, taskImplSel);
    if (taskImpl) {
      BOOL added = class_addMethod(cls, taskChallengeSel,
                                   method_getImplementation(taskImpl),
                                   method_getTypeEncoding(taskImpl));
      if (added) {
        NSLog(@"[BoardingPass TLS] Injected task-level challenge handler");
      } else {
        NSLog(@"[BoardingPass TLS] Task-level handler exists, swizzling");
        Method original = class_getInstanceMethod(cls, taskChallengeSel);
        if (original) method_exchangeImplementations(original, taskImpl);
      }
    }
  });
}

/**
 * Session-level challenge handler (URLSession:didReceiveChallenge:completionHandler:)
 * This is what NSURLSession calls for TLS server trust challenges.
 */
- (void)bp_URLSession:(NSURLSession *)session
  didReceiveChallenge:(NSURLAuthenticationChallenge *)challenge
    completionHandler:(void (^)(NSURLSessionAuthChallengeDisposition, NSURLCredential *_Nullable))completionHandler
{
  BPHandleChallenge(challenge, completionHandler);
}

/**
 * Task-level challenge handler (URLSession:task:didReceiveChallenge:completionHandler:)
 * Fallback for any challenges not handled at the session level.
 */
- (void)bp_URLSession:(NSURLSession *)session
                 task:(NSURLSessionTask *)task
  didReceiveChallenge:(NSURLAuthenticationChallenge *)challenge
    completionHandler:(void (^)(NSURLSessionAuthChallengeDisposition, NSURLCredential *_Nullable))completionHandler
{
  BPHandleChallenge(challenge, completionHandler);
}

@end
