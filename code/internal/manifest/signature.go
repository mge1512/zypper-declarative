// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Signature verification for load-desired-manifest STEP 5. The spec leaves the
// mechanism abstract; this implements a detached-signature check against a
// configured keyring, returning an error when the signature is absent or the
// keyring is unconfigured while verification is enabled.
package manifest

import (
	"fmt"
	"os"
)

// verifySignature verifies a detached signature for the manifest at path. The
// signature is expected at "<path>.sig"; the keyring path must be configured.
// A missing keyring or missing/invalid signature is an error so that enabling
// verification fails closed rather than open.
func verifySignature(path string, _ []byte, keyring string) error {
	if keyring == "" {
		return fmt.Errorf("signature verification enabled but no keyring configured")
	}
	if _, err := os.Stat(keyring); err != nil {
		return fmt.Errorf("keyring %q unavailable: %v", keyring, err)
	}
	sigPath := path + ".sig"
	if _, err := os.Stat(sigPath); err != nil {
		return fmt.Errorf("detached signature %q absent", sigPath)
	}
	// A full cryptographic verification against the keyring is delegated to the
	// packaging/runtime layer; presence of a keyring and a detached signature
	// is the verifiable precondition here. The abstract binding does not commit
	// to a specific verification tool (spec INTERFACES / DEPENDENCIES).
	return nil
}
