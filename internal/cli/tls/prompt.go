package tls

import (
	"bufio"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"github.com/fzdarsky/boardingpass/internal/cli/clicontext"
)

// PromptAcceptCertificate prompts the user to accept or reject an unknown certificate.
// Returns true if the user accepts, false if they reject.
// If --assumeyes flag is set, automatically accepts the certificate without prompting.
func PromptAcceptCertificate(host string, cert *x509.Certificate) bool {
	fingerprint := ComputeFingerprint(cert)

	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "WARNING: Unknown TLS certificate\n")
	fmt.Fprintf(os.Stderr, "  Host:        %s\n", host)
	fmt.Fprintf(os.Stderr, "  Subject:     %s\n", cert.Subject)
	fmt.Fprintf(os.Stderr, "  Issuer:      %s\n", cert.Issuer)
	fmt.Fprintf(os.Stderr, "  Valid From:  %s\n", cert.NotBefore)
	fmt.Fprintf(os.Stderr, "  Valid Until: %s\n", cert.NotAfter)
	fmt.Fprintf(os.Stderr, "  Fingerprint: %s\n", fingerprint)
	fmt.Fprintf(os.Stderr, "\n")

	// Check if assumeyes flag is set
	if clicontext.AssumeYes() {
		fmt.Fprintf(os.Stderr, "Automatically accepting certificate (--assumeyes flag is set)\n")
		return true
	}

	return promptYesNo("Do you want to accept this certificate?")
}

// promptYesNo prompts the user for a yes/no answer.
// Returns true for yes, false for no.
func promptYesNo(question string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Fprintf(os.Stderr, "%s (yes/no): ", question)

		response, err := reader.ReadString('\n')
		if err != nil {
			return false
		}

		response = strings.ToLower(strings.TrimSpace(response))

		switch response {
		case "yes", "y":
			return true
		case "no", "n":
			return false
		default:
			fmt.Fprintf(os.Stderr, "Please answer 'yes' or 'no'\n")
		}
	}
}
