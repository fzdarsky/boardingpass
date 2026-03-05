import ExpoModulesCore
import Security
import CommonCrypto

/// UserDefaults key shared between this module and the TLS override.
/// The TLS override (RCTHTTPSOverride.m) reads from this store to decide
/// whether to accept a server's self-signed certificate.
private let pinStoreKey = "com.boardingpass.certificatePins"

public class CertificatePinningModule: Module {
  public func definition() -> ModuleDefinition {
    Name("CertificatePinning")

    OnCreate {
      Self.injectTLSOverride()

      // RCTHTTPRequestHandler lazily creates its NSURLSession during JS bundle
      // loading, which happens BEFORE Expo module OnCreate.  That session caches
      // respondsToSelector: results, so our freshly-injected challenge handler is
      // invisible to it.  Invalidating the session on the next run-loop iteration
      // forces recreation, picking up the newly-added delegate methods.
      DispatchQueue.main.async {
        let result = Self.invalidateRCTHTTPSession()
        NSLog("[BoardingPass TLS] Post-injection session invalidation: %@", "\(result)")
      }
    }

    /// Fetch the TLS certificate from a remote server.
    ///
    /// Opens a TLS connection, captures the server certificate (even self-signed),
    /// computes the SHA-256 fingerprint, and returns certificate metadata.
    AsyncFunction("fetchServerCertificate") {
      (host: String, port: Int) -> [String: Any] in
      return try await fetchCertificate(host: host, port: port)
    }

    /// Store a certificate fingerprint for a host, enabling the TLS override
    /// to accept that certificate on future HTTPS connections.
    Function("pinCertificate") { (hostKey: String, fingerprint: String) in
      Self.pinCertificateSync(hostKey: hostKey, fingerprint: fingerprint)
    }

    /// Remove a pinned certificate for a host.
    Function("unpinCertificate") { (hostKey: String) in
      Self.unpinCertificateSync(hostKey: hostKey)
    }

    /// Get the pinned fingerprint for a host, or nil if none is pinned.
    Function("getPinnedFingerprint") { (hostKey: String) -> String? in
      return Self.getPinnedFingerprintSync(hostKey: hostKey)
    }

    /// Clear all pinned certificates.
    Function("clearAllPins") {
      Self.clearAllPinsSync()
    }

    /// Check if the TLS challenge handler was successfully injected into RCTHTTPRequestHandler.
    Function("isTLSOverrideActive") { () -> Bool in
      guard let cls = objc_getClass("RCTHTTPRequestHandler") as? AnyClass else { return false }
      let sel = NSSelectorFromString("URLSession:didReceiveChallenge:completionHandler:")
      return class_getInstanceMethod(cls, sel) != nil
    }

    /// Return the number of times our challenge handler has been called.
    /// Used for diagnostics — check before and after a request to see if the
    /// handler fired for RCT's session (vs only for our diagnostic session).
    Function("getChallengeCount") { () -> Int in
      return Self.challengeCallCount
    }

    /// Reset the challenge call counter (call before a request to measure).
    Function("resetChallengeCount") {
      Self.challengeCallCount = 0
    }

    /// Return the full contents of the native pin store (UserDefaults).
    /// Used for diagnostics — verifies pins are stored with correct keys.
    Function("getPinStore") { () -> [String: String] in
      return Self.getPinStore()
    }

    /// Test HTTPS connectivity to a host using a fresh NSURLSession
    /// with our TOFU challenge handler. Returns diagnostic info about
    /// whether the TLS handshake succeeds and whether pins are matched.
    AsyncFunction("diagnoseTLS") { (host: String, port: Int) -> [String: Any] in
      return await Self.diagnoseTLS(host: host, port: port)
    }

    /// Invalidate RCTHTTPRequestHandler's cached NSURLSession.
    ///
    /// NSURLSession caches respondsToSelector: results at creation time.
    /// If the session was created before our challenge handler was injected,
    /// it won't call our handler. Invalidating forces recreation of the
    /// session, which picks up the dynamically added method.
    ///
    /// Also useful after pinning a certificate: the old session may have
    /// cached a TLS failure from a previous handshake.
    Function("invalidateHTTPSession") { () -> [String: Any] in
      return Self.invalidateRCTHTTPSession()
    }

    /// Perform an HTTPS request using a native URLSession with TOFU certificate
    /// handling. Bypasses RCTHTTPRequestHandler entirely to avoid its session
    /// caching issues with dynamically injected TLS challenge handlers.
    ///
    /// This is the primary mechanism for making HTTPS requests to BoardingPass
    /// devices with self-signed certificates.
    AsyncFunction("nativeFetch") {
      (urlString: String, method: String, headers: [String: Any], body: String, timeoutMs: Double) -> [String: Any] in
      return await Self.nativeFetch(urlString: urlString, method: method, headers: headers, body: body, timeoutMs: timeoutMs)
    }
  }

  /// Counter for challenge handler invocations (diagnostic use).
  private static var challengeCallCount = 0

  // MARK: - TLS Override Injection

  /// Inject session-level and task-level authentication challenge handlers into
  /// RCTHTTPRequestHandler so that NSURLSession accepts self-signed certificates
  /// according to TOFU rules.
  ///
  /// This is called from OnCreate to guarantee execution (the ObjC category's
  /// +load may not fire reliably depending on linker/load order).
  private static func injectTLSOverride() {
    guard let cls = objc_getClass("RCTHTTPRequestHandler") as? AnyClass else {
      NSLog("[BoardingPass TLS] ERROR: RCTHTTPRequestHandler class not found")
      return
    }

    // Session-level handler (what NSURLSession calls for server trust challenges)
    let sessionSel = NSSelectorFromString("URLSession:didReceiveChallenge:completionHandler:")
    let sessionBlock: @convention(block) (
      Any, URLSession, URLAuthenticationChallenge,
      @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void
    ) -> Void = { _, _, challenge, completionHandler in
      Self.handleTLSChallenge(challenge, completionHandler: completionHandler)
    }
    let sessionIMP = imp_implementationWithBlock(sessionBlock)
    // Type encoding: v = void, @ = object (self), : = SEL, @ = session, @ = challenge, @? = block
    if class_addMethod(cls, sessionSel, sessionIMP, "v@:@@@?") {
      NSLog("[BoardingPass TLS] Injected session-level challenge handler (from Swift)")
    } else {
      NSLog("[BoardingPass TLS] Session-level handler already exists")
    }

    // Task-level handler (fallback)
    let taskSel = NSSelectorFromString("URLSession:task:didReceiveChallenge:completionHandler:")
    let taskBlock: @convention(block) (
      Any, URLSession, URLSessionTask, URLAuthenticationChallenge,
      @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void
    ) -> Void = { _, _, _, challenge, completionHandler in
      Self.handleTLSChallenge(challenge, completionHandler: completionHandler)
    }
    let taskIMP = imp_implementationWithBlock(taskBlock)
    if class_addMethod(cls, taskSel, taskIMP, "v@:@@@@?") {
      NSLog("[BoardingPass TLS] Injected task-level challenge handler (from Swift)")
    } else {
      NSLog("[BoardingPass TLS] Task-level handler already exists")
    }
  }

  /// Reset RCTHTTPRequestHandler's cached NSURLSession so it picks up
  /// the dynamically added TLS challenge handler on next request.
  ///
  /// IMPORTANT: We cannot call RCTHTTPRequestHandler's `-invalidate` method
  /// because it marks the handler as permanently dead (isValid → false).
  /// Instead, we directly nil the `_session` and `_delegates` ivars via the
  /// ObjC runtime, which resets isValid → true and lets `sendRequest:` lazily
  /// recreate the session with our injected challenge handler methods.
  private static func invalidateRCTHTTPSession() -> [String: Any] {
    guard let cls = objc_getClass("RCTHTTPRequestHandler") as? AnyClass else {
      NSLog("[BoardingPass TLS] invalidateHTTPSession: RCTHTTPRequestHandler class not found")
      return ["success": false, "error": "RCTHTTPRequestHandler class not found"]
    }

    // Find the instance through RCTBridge's module registry
    guard let bridgeClass = NSClassFromString("RCTBridge") else {
      NSLog("[BoardingPass TLS] invalidateHTTPSession: RCTBridge class not found")
      return ["success": false, "error": "RCTBridge class not found"]
    }

    guard let currentBridge = bridgeClass.value(forKeyPath: "currentBridge") as? NSObject else {
      NSLog("[BoardingPass TLS] invalidateHTTPSession: No current bridge")
      return ["success": false, "error": "No current RCTBridge"]
    }

    let moduleForClassSel = NSSelectorFromString("moduleForClass:")
    guard currentBridge.responds(to: moduleForClassSel) else {
      NSLog("[BoardingPass TLS] invalidateHTTPSession: Bridge doesn't respond to moduleForClass:")
      return ["success": false, "error": "Bridge API mismatch"]
    }

    guard let handlerObj = currentBridge.perform(moduleForClassSel, with: cls)?
            .takeUnretainedValue() as? NSObject else {
      NSLog("[BoardingPass TLS] invalidateHTTPSession: Could not get handler instance")
      return ["success": false, "error": "Handler instance not found"]
    }

    // Access _session ivar directly: invalidateAndCancel the old session, then nil it
    guard let sessionIvar = class_getInstanceVariable(cls, "_session") else {
      NSLog("[BoardingPass TLS] invalidateHTTPSession: _session ivar not found")
      return ["success": false, "error": "_session ivar not found"]
    }

    if let oldSession = object_getIvar(handlerObj, sessionIvar) as? URLSession {
      oldSession.invalidateAndCancel()
    }
    object_setIvar(handlerObj, sessionIvar, nil)

    // Also nil _delegates so isValid returns true (allows session recreation)
    if let delegatesIvar = class_getInstanceVariable(cls, "_delegates") {
      object_setIvar(handlerObj, delegatesIvar, nil)
    }

    NSLog("[BoardingPass TLS] invalidateHTTPSession: Session reset — will recreate on next request")

    return ["success": true, "method": "ivar_reset"]
  }

  /// Shared challenge handling logic for TOFU certificate pinning.
  private static func handleTLSChallenge(
    _ challenge: URLAuthenticationChallenge,
    completionHandler: @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void
  ) {
    challengeCallCount += 1
    NSLog("[BoardingPass TLS] Challenge #%d: %@ for %@:%d",
          challengeCallCount,
          challenge.protectionSpace.authenticationMethod,
          challenge.protectionSpace.host,
          challenge.protectionSpace.port)

    guard challenge.protectionSpace.authenticationMethod == NSURLAuthenticationMethodServerTrust,
          let trust = challenge.protectionSpace.serverTrust else {
      completionHandler(.performDefaultHandling, nil)
      return
    }

    // Compute SHA-256 fingerprint of the leaf certificate
    var leafCert: SecCertificate?
    if #available(iOS 15.0, *) {
      if let chain = SecTrustCopyCertificateChain(trust) as? [SecCertificate], let first = chain.first {
        leafCert = first
      }
    } else {
      if SecTrustGetCertificateCount(trust) > 0 {
        leafCert = SecTrustGetCertificateAtIndex(trust, 0)
      }
    }

    guard let cert = leafCert else {
      completionHandler(.cancelAuthenticationChallenge, nil)
      return
    }

    let derData = SecCertificateCopyData(cert) as Data
    var digest = [UInt8](repeating: 0, count: Int(CC_SHA256_DIGEST_LENGTH))
    _ = derData.withUnsafeBytes { CC_SHA256($0.baseAddress, CC_LONG(derData.count), &digest) }
    let fingerprint = digest.map { String(format: "%02x", $0) }.joined()

    // Look up pin store
    let pins = UserDefaults.standard.dictionary(forKey: pinStoreKey) as? [String: String] ?? [:]
    let host = challenge.protectionSpace.host
    let port = challenge.protectionSpace.port
    let hostPortKey = "\(host):\(port)"
    let pinnedFingerprint = pins[hostPortKey] ?? pins[host]

    if let pinned = pinnedFingerprint {
      if fingerprint == pinned.lowercased() {
        NSLog("[BoardingPass TLS] Pin match for %@:%d", host, port)
        completionHandler(.useCredential, URLCredential(trust: trust))
      } else {
        NSLog("[BoardingPass TLS] Pin MISMATCH for %@:%d — rejecting", host, port)
        completionHandler(.cancelAuthenticationChallenge, nil)
      }
    } else {
      // TOFU: no pin exists, accept and let JS layer handle trust dialog
      NSLog("[BoardingPass TLS] TOFU: accepting unpinned cert for %@:%d (fingerprint: %@)",
            host, port, fingerprint)
      completionHandler(.useCredential, URLCredential(trust: trust))
    }
  }

  // MARK: - TLS Diagnostics

  /// Test HTTPS connectivity using a fresh NSURLSession with our challenge handler.
  /// This bypasses RCTHTTPRequestHandler to verify TLS handling works at the native level.
  private static func diagnoseTLS(host: String, port: Int) async -> [String: Any] {
    let delegate = DiagnosticTLSDelegate()
    let session = URLSession(
      configuration: .ephemeral,
      delegate: delegate,
      delegateQueue: nil
    )

    guard let url = URL(string: "https://\(host):\(port)/api/v1/info") else {
      return ["success": false, "error": "Invalid URL"]
    }

    var request = URLRequest(url: url, timeoutInterval: 10)
    request.httpMethod = "GET"

    do {
      let (data, response) = try await session.data(for: request)
      let httpResponse = response as? HTTPURLResponse
      session.invalidateAndCancel()

      return [
        "success": true,
        "statusCode": httpResponse?.statusCode ?? 0,
        "bodyLength": data.count,
        "challengeHandlerCalled": delegate.challengeHandlerCalled,
        "challengeHost": delegate.challengeHost ?? "",
        "challengePort": delegate.challengePort,
        "decision": delegate.decision,
        "pinStore": getPinStore(),
      ]
    } catch {
      session.invalidateAndCancel()
      return [
        "success": false,
        "error": error.localizedDescription,
        "challengeHandlerCalled": delegate.challengeHandlerCalled,
        "challengeHost": delegate.challengeHost ?? "",
        "challengePort": delegate.challengePort,
        "decision": delegate.decision,
        "pinStore": getPinStore(),
      ]
    }
  }

  // MARK: - Native HTTPS Fetch

  /// Perform an HTTPS request using a fresh ephemeral URLSession with TOFU
  /// certificate handling.  This bypasses RCTHTTPRequestHandler entirely,
  /// solving the issue where its NSURLSession ignores dynamically injected
  /// TLS challenge handlers.
  ///
  /// Uses the same TOFU delegate as `diagnoseTLS` (which is proven to work).
  private static func nativeFetch(
    urlString: String,
    method: String,
    headers: [String: Any],
    body: String,
    timeoutMs: Double
  ) async -> [String: Any] {
    guard let url = URL(string: urlString) else {
      return ["error": "Invalid URL: \(urlString)", "status": 0, "code": "ERR_BAD_REQUEST"]
    }

    NSLog("[BoardingPass nativeFetch] %@ %@", method.uppercased(), urlString)

    let delegate = TOFUDelegate()
    let session = URLSession(
      configuration: .ephemeral,
      delegate: delegate,
      delegateQueue: nil
    )

    var request = URLRequest(url: url, timeoutInterval: timeoutMs / 1000.0)
    request.httpMethod = method.uppercased()

    for (key, value) in headers {
      // Expo Modules bridges JS strings as Optional<String> inside Any.
      // String interpolation "\(value)" on Any wrapping Optional produces
      // "Optional(...)" — use 'as? String' to unwrap cleanly.
      if let stringValue = value as? String {
        request.setValue(stringValue, forHTTPHeaderField: key)
      }
    }

    if !body.isEmpty {
      request.httpBody = body.data(using: .utf8)
    }

    do {
      let (data, response) = try await session.data(for: request)
      let httpResponse = response as? HTTPURLResponse
      session.invalidateAndCancel()

      var responseHeaders: [String: String] = [:]
      httpResponse?.allHeaderFields.forEach { key, value in
        responseHeaders["\(key)"] = "\(value)"
      }

      let bodyString = String(data: data, encoding: .utf8) ?? ""

      NSLog("[BoardingPass nativeFetch] Response: %d (%d bytes)",
            httpResponse?.statusCode ?? 0, data.count)

      return [
        "status": httpResponse?.statusCode ?? 0,
        "headers": responseHeaders,
        "body": bodyString,
      ]
    } catch {
      session.invalidateAndCancel()

      let nsError = error as NSError
      var errorCode = "ERR_NETWORK"
      if nsError.code == NSURLErrorTimedOut {
        errorCode = "ECONNABORTED"
      } else if nsError.code == NSURLErrorCannotConnectToHost {
        errorCode = "ECONNREFUSED"
      } else if nsError.code == NSURLErrorNotConnectedToInternet {
        errorCode = "ENETUNREACH"
      }

      NSLog("[BoardingPass nativeFetch] Error: %@ (code: %@)",
            error.localizedDescription, errorCode)

      return [
        "error": error.localizedDescription,
        "status": 0,
        "code": errorCode,
      ]
    }
  }

  // MARK: - Certificate Fetching

  /// Perform a TLS handshake with the server and extract its certificate.
  private func fetchCertificate(host: String, port: Int) async throws -> [String: Any] {
    let delegate = CertificateFetchDelegate()
    let session = URLSession(
      configuration: .ephemeral,
      delegate: delegate,
      delegateQueue: nil
    )

    guard let url = URL(string: "https://\(host):\(port)/") else {
      throw CertPinError.invalidHost(host, port)
    }

    var request = URLRequest(url: url, timeoutInterval: 10)
    request.httpMethod = "HEAD"

    // The request may fail after the TLS handshake (404, connection reset, etc.)
    // That's fine — we only need the certificate from the handshake.
    do {
      let _ = try await session.data(for: request)
    } catch {
      if delegate.certificate == nil {
        throw CertPinError.fetchFailed(host, port, error.localizedDescription)
      }
    }

    session.invalidateAndCancel()

    guard let cert = delegate.certificate else {
      throw CertPinError.noCertificate(host, port)
    }

    return extractCertificateInfo(from: cert)
  }

  /// Extract metadata from a SecCertificate.
  private func extractCertificateInfo(from cert: SecCertificate) -> [String: Any] {
    let derData = SecCertificateCopyData(cert) as Data

    // SHA-256 fingerprint
    var digest = [UInt8](repeating: 0, count: Int(CC_SHA256_DIGEST_LENGTH))
    _ = derData.withUnsafeBytes {
      CC_SHA256($0.baseAddress, CC_LONG(derData.count), &digest)
    }
    let fingerprint = digest.map { String(format: "%02x", $0) }.joined()

    // Subject summary (CN or fallback)
    let subject = (SecCertificateCopySubjectSummary(cert) as String?) ?? "Unknown"

    // Self-signed detection: compare normalized subject and issuer DER sequences
    let subjectSeq = SecCertificateCopyNormalizedSubjectSequence(cert) as Data?
    let issuerSeq = SecCertificateCopyNormalizedIssuerSequence(cert) as Data?
    let isSelfSigned = (subjectSeq != nil && issuerSeq != nil && subjectSeq == issuerSeq)

    // Issuer: for self-signed certs, same as subject
    let issuer = isSelfSigned ? subject : "Unknown CA"

    // PEM encoding
    let base64 = derData.base64EncodedString(options: [
      .lineLength64Characters, .endLineWithLineFeed
    ])
    let pem = "-----BEGIN CERTIFICATE-----\n\(base64)\n-----END CERTIFICATE-----"

    // Validity dates (best effort from DER)
    let dates = extractValidityDates(from: derData)

    return [
      "fingerprint": fingerprint,
      "subject": subject,
      "issuer": issuer,
      "isSelfSigned": isSelfSigned,
      "pemEncoded": pem,
      "validFrom": dates.notBefore ?? "",
      "validTo": dates.notAfter ?? "",
    ]
  }

  /// Best-effort extraction of notBefore/notAfter dates from DER-encoded certificate.
  /// Searches for UTCTime (tag 0x17) or GeneralizedTime (tag 0x18) fields.
  private func extractValidityDates(from data: Data) -> (notBefore: String?, notAfter: String?) {
    let bytes = [UInt8](data)
    var dates: [String] = []

    var i = 0
    while i < bytes.count - 2 && dates.count < 2 {
      let tag = bytes[i]
      // UTCTime (0x17) or GeneralizedTime (0x18)
      if tag == 0x17 || tag == 0x18 {
        let length = Int(bytes[i + 1])
        let start = i + 2
        if start + length <= bytes.count {
          if let dateStr = String(bytes: Array(bytes[start..<start+length]), encoding: .ascii) {
            let isoDate = parseASN1Date(dateStr, isUTC: tag == 0x17)
            if let isoDate = isoDate {
              dates.append(isoDate)
            }
          }
        }
        i = start + length
      } else {
        i += 1
      }
    }

    return (
      notBefore: dates.count > 0 ? dates[0] : nil,
      notAfter: dates.count > 1 ? dates[1] : nil
    )
  }

  /// Parse ASN.1 UTCTime or GeneralizedTime to ISO 8601 string.
  private func parseASN1Date(_ s: String, isUTC: Bool) -> String? {
    let formatter = DateFormatter()
    formatter.locale = Locale(identifier: "en_US_POSIX")
    formatter.timeZone = TimeZone(identifier: "UTC")

    if isUTC {
      // UTCTime: YYMMDDHHMMSSZ
      formatter.dateFormat = "yyMMddHHmmss'Z'"
    } else {
      // GeneralizedTime: YYYYMMDDHHMMSSZ
      formatter.dateFormat = "yyyyMMddHHmmss'Z'"
    }

    guard let date = formatter.date(from: s) else { return nil }

    let iso = ISO8601DateFormatter()
    return iso.string(from: date)
  }

  // MARK: - Pin Store (shared with TLS override)

  static func pinCertificateSync(hostKey: String, fingerprint: String) {
    var pins = getPinStore()
    pins[hostKey] = fingerprint.lowercased()
    savePinStore(pins)
  }

  static func unpinCertificateSync(hostKey: String) {
    var pins = getPinStore()
    pins.removeValue(forKey: hostKey)
    savePinStore(pins)
  }

  static func getPinnedFingerprintSync(hostKey: String) -> String? {
    let pins = getPinStore()
    return pins[hostKey]
  }

  static func clearAllPinsSync() {
    savePinStore([:])
  }

  static func getPinStore() -> [String: String] {
    return UserDefaults.standard.dictionary(forKey: pinStoreKey) as? [String: String] ?? [:]
  }

  private static func savePinStore(_ pins: [String: String]) {
    UserDefaults.standard.set(pins, forKey: pinStoreKey)
    UserDefaults.standard.synchronize()
  }
}

// MARK: - Certificate Fetch Delegate

/// URLSession delegate that captures the server certificate during TLS handshake.
/// Accepts ALL certificates (including self-signed) because the purpose is inspection,
/// not validation. Validation happens at the JS layer via the TOFU flow.
private class CertificateFetchDelegate: NSObject, URLSessionDelegate {
  var certificate: SecCertificate?

  func urlSession(
    _ session: URLSession,
    didReceive challenge: URLAuthenticationChallenge,
    completionHandler: @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void
  ) {
    guard challenge.protectionSpace.authenticationMethod == NSURLAuthenticationMethodServerTrust,
          let trust = challenge.protectionSpace.serverTrust else {
      completionHandler(.performDefaultHandling, nil)
      return
    }

    // Capture the leaf certificate from the chain
    if #available(iOS 15.0, *) {
      if let chain = SecTrustCopyCertificateChain(trust) as? [SecCertificate],
         let cert = chain.first {
        self.certificate = cert
      }
    } else {
      // Fallback for iOS < 15 (deprecated API)
      if SecTrustGetCertificateCount(trust) > 0,
         let cert = SecTrustGetCertificateAtIndex(trust, 0) {
        self.certificate = cert
      }
    }

    // Accept any certificate (we're inspecting, not validating)
    completionHandler(.useCredential, URLCredential(trust: trust))
  }
}

// MARK: - Diagnostic TLS Delegate

/// URLSession delegate for TLS diagnostics. Uses the same TOFU logic
/// as the injected handler but captures diagnostic info for JS consumption.
private class DiagnosticTLSDelegate: NSObject, URLSessionDelegate {
  var challengeHandlerCalled = false
  var challengeHost: String?
  var challengePort: Int = 0
  var decision: String = "not_called"

  func urlSession(
    _ session: URLSession,
    didReceive challenge: URLAuthenticationChallenge,
    completionHandler: @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void
  ) {
    challengeHandlerCalled = true
    challengeHost = challenge.protectionSpace.host
    challengePort = challenge.protectionSpace.port

    guard challenge.protectionSpace.authenticationMethod == NSURLAuthenticationMethodServerTrust,
          let trust = challenge.protectionSpace.serverTrust else {
      decision = "default_handling"
      completionHandler(.performDefaultHandling, nil)
      return
    }

    // Compute SHA-256 fingerprint (same logic as handleTLSChallenge)
    var leafCert: SecCertificate?
    if #available(iOS 15.0, *) {
      if let chain = SecTrustCopyCertificateChain(trust) as? [SecCertificate], let first = chain.first {
        leafCert = first
      }
    } else {
      if SecTrustGetCertificateCount(trust) > 0 {
        leafCert = SecTrustGetCertificateAtIndex(trust, 0)
      }
    }

    guard let cert = leafCert else {
      decision = "no_leaf_cert"
      completionHandler(.cancelAuthenticationChallenge, nil)
      return
    }

    let derData = SecCertificateCopyData(cert) as Data
    var digest = [UInt8](repeating: 0, count: Int(CC_SHA256_DIGEST_LENGTH))
    _ = derData.withUnsafeBytes { CC_SHA256($0.baseAddress, CC_LONG(derData.count), &digest) }
    let fingerprint = digest.map { String(format: "%02x", $0) }.joined()

    // Check pin store
    let pins = UserDefaults.standard.dictionary(forKey: pinStoreKey) as? [String: String] ?? [:]
    let host = challenge.protectionSpace.host
    let port = challenge.protectionSpace.port
    let hostPortKey = "\(host):\(port)"
    let pinnedFingerprint = pins[hostPortKey] ?? pins[host]

    if let pinned = pinnedFingerprint {
      if fingerprint == pinned.lowercased() {
        decision = "pin_match(\(hostPortKey))"
        completionHandler(.useCredential, URLCredential(trust: trust))
      } else {
        decision = "pin_mismatch(\(hostPortKey),got=\(fingerprint.prefix(16))...,expected=\(pinned.prefix(16))...)"
        completionHandler(.cancelAuthenticationChallenge, nil)
      }
    } else {
      decision = "tofu_accept(\(hostPortKey),fingerprint=\(fingerprint.prefix(16))...)"
      completionHandler(.useCredential, URLCredential(trust: trust))
    }
  }
}

// MARK: - TOFU Delegate (for nativeFetch)

/// Lightweight URLSession delegate that implements TOFU certificate pinning.
/// Used by `nativeFetch` to handle self-signed certificates.  Same logic
/// as `handleTLSChallenge` but without the diagnostic counter.
private class TOFUDelegate: NSObject, URLSessionDelegate {
  func urlSession(
    _ session: URLSession,
    didReceive challenge: URLAuthenticationChallenge,
    completionHandler: @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void
  ) {
    guard challenge.protectionSpace.authenticationMethod == NSURLAuthenticationMethodServerTrust,
          let trust = challenge.protectionSpace.serverTrust else {
      completionHandler(.performDefaultHandling, nil)
      return
    }

    // Extract leaf certificate
    var leafCert: SecCertificate?
    if #available(iOS 15.0, *) {
      if let chain = SecTrustCopyCertificateChain(trust) as? [SecCertificate], let first = chain.first {
        leafCert = first
      }
    } else {
      if SecTrustGetCertificateCount(trust) > 0 {
        leafCert = SecTrustGetCertificateAtIndex(trust, 0)
      }
    }

    guard let cert = leafCert else {
      completionHandler(.cancelAuthenticationChallenge, nil)
      return
    }

    // Compute SHA-256 fingerprint of DER-encoded certificate
    let derData = SecCertificateCopyData(cert) as Data
    var digest = [UInt8](repeating: 0, count: Int(CC_SHA256_DIGEST_LENGTH))
    _ = derData.withUnsafeBytes { CC_SHA256($0.baseAddress, CC_LONG(derData.count), &digest) }
    let fingerprint = digest.map { String(format: "%02x", $0) }.joined()

    // Look up pin store
    let pins = UserDefaults.standard.dictionary(forKey: pinStoreKey) as? [String: String] ?? [:]
    let host = challenge.protectionSpace.host
    let port = challenge.protectionSpace.port
    let hostPortKey = "\(host):\(port)"
    let pinnedFingerprint = pins[hostPortKey] ?? pins[host]

    if let pinned = pinnedFingerprint {
      if fingerprint == pinned.lowercased() {
        NSLog("[BoardingPass TLS] nativeFetch: pin match for %@:%d", host, port)
        completionHandler(.useCredential, URLCredential(trust: trust))
      } else {
        NSLog("[BoardingPass TLS] nativeFetch: pin MISMATCH for %@:%d", host, port)
        completionHandler(.cancelAuthenticationChallenge, nil)
      }
    } else {
      // TOFU: no pin, accept and let JS layer handle trust dialog
      NSLog("[BoardingPass TLS] nativeFetch: TOFU accept for %@:%d", host, port)
      completionHandler(.useCredential, URLCredential(trust: trust))
    }
  }
}

// MARK: - Errors

private enum CertPinError: LocalizedError {
  case invalidHost(String, Int)
  case fetchFailed(String, Int, String)
  case noCertificate(String, Int)

  var errorDescription: String? {
    switch self {
    case .invalidHost(let host, let port):
      return "Invalid host: \(host):\(port)"
    case .fetchFailed(let host, let port, let reason):
      return "Failed to fetch certificate from \(host):\(port): \(reason)"
    case .noCertificate(let host, let port):
      return "No certificate received from \(host):\(port)"
    }
  }
}
