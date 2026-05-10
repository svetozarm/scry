package update

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
)

//go:embed cosign.pub
var cosignPubKey []byte

// VerifySignature verifies that sigPath contains a valid cosign signature
// for the file at artifactPath, using the embedded public key.
func VerifySignature(artifactPath, sigPath string) error {
	pubKey, err := parsePublicKey(cosignPubKey)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSignatureInvalid, err)
	}

	artifact, err := os.ReadFile(artifactPath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSignatureInvalid, err)
	}

	sigB64, err := os.ReadFile(sigPath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSignatureInvalid, err)
	}

	sig, err := base64.StdEncoding.DecodeString(string(bytes.TrimSpace(sigB64)))
	if err != nil {
		return fmt.Errorf("%w: invalid base64 signature: %v", ErrSignatureInvalid, err)
	}

	digest := sha256.Sum256(artifact)

	if !ecdsaVerify(pubKey, digest[:], sig) {
		return ErrSignatureInvalid
	}

	return nil
}

func parsePublicKey(pemBytes []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	ecKey, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not ECDSA")
	}
	return ecKey, nil
}

func ecdsaVerify(pub *ecdsa.PublicKey, digest, sig []byte) bool {
	// cosign produces DER-encoded ECDSA signatures
	// Try DER first, then raw r||s
	if ecdsa.VerifyASN1(pub, digest, sig) {
		return true
	}

	// Fallback: raw r||s (each half the key size)
	keySize := (pub.Params().BitSize + 7) / 8
	if len(sig) == 2*keySize {
		r := new(big.Int).SetBytes(sig[:keySize])
		s := new(big.Int).SetBytes(sig[keySize:])
		return ecdsa.Verify(pub, digest, r, s)
	}

	return false
}

// HashForSigning returns the SHA-256 hash algorithm used by cosign.
func HashForSigning() crypto.Hash {
	return crypto.SHA256
}
