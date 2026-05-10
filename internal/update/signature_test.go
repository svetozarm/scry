package update

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifySignature_Valid(t *testing.T) {
	privKey, pubPEM := generateTestKeyPair(t)
	artifact := []byte("checksums content here")
	sig := signBlob(t, privKey, artifact)

	dir := t.TempDir()
	artifactPath := filepath.Join(dir, "checksums.txt")
	sigPath := filepath.Join(dir, "checksums.txt.sig")

	require.NoError(t, os.WriteFile(artifactPath, artifact, 0644))
	require.NoError(t, os.WriteFile(sigPath, []byte(base64.StdEncoding.EncodeToString(sig)), 0644))

	// Override embedded key for test
	origKey := cosignPubKey
	cosignPubKey = pubPEM
	defer func() { cosignPubKey = origKey }()

	err := VerifySignature(artifactPath, sigPath)
	assert.NoError(t, err)
}

func TestVerifySignature_InvalidSignature(t *testing.T) {
	_, pubPEM := generateTestKeyPair(t)
	artifact := []byte("checksums content here")

	dir := t.TempDir()
	artifactPath := filepath.Join(dir, "checksums.txt")
	sigPath := filepath.Join(dir, "checksums.txt.sig")

	require.NoError(t, os.WriteFile(artifactPath, artifact, 0644))
	require.NoError(t, os.WriteFile(sigPath, []byte(base64.StdEncoding.EncodeToString([]byte("badsig"))), 0644))

	origKey := cosignPubKey
	cosignPubKey = pubPEM
	defer func() { cosignPubKey = origKey }()

	err := VerifySignature(artifactPath, sigPath)
	assert.ErrorIs(t, err, ErrSignatureInvalid)
}

func TestVerifySignature_TamperedArtifact(t *testing.T) {
	privKey, pubPEM := generateTestKeyPair(t)
	artifact := []byte("original content")
	sig := signBlob(t, privKey, artifact)

	dir := t.TempDir()
	artifactPath := filepath.Join(dir, "checksums.txt")
	sigPath := filepath.Join(dir, "checksums.txt.sig")

	// Write tampered content
	require.NoError(t, os.WriteFile(artifactPath, []byte("tampered content"), 0644))
	require.NoError(t, os.WriteFile(sigPath, []byte(base64.StdEncoding.EncodeToString(sig)), 0644))

	origKey := cosignPubKey
	cosignPubKey = pubPEM
	defer func() { cosignPubKey = origKey }()

	err := VerifySignature(artifactPath, sigPath)
	assert.ErrorIs(t, err, ErrSignatureInvalid)
}

func generateTestKeyPair(t *testing.T) (*ecdsa.PrivateKey, []byte) {
	t.Helper()
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	pubDER, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	require.NoError(t, err)

	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	return privKey, pubPEM
}

func signBlob(t *testing.T, key *ecdsa.PrivateKey, data []byte) []byte {
	t.Helper()
	digest := sha256.Sum256(data)
	sig, err := ecdsa.SignASN1(rand.Reader, key, digest[:])
	require.NoError(t, err)
	return sig
}
