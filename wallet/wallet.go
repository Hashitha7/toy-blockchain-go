// Package wallet provides ed25519 digital signature generation and verification.
package wallet

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateKeyPair creates a new ed25519 public/private key pair and returns
// them as hex-encoded strings.
func GenerateKeyPair() (pubKeyHex string, privKeyHex string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key: %w", err)
	}
	return hex.EncodeToString(pub), hex.EncodeToString(priv), nil
}

// Sign takes a hex-encoded private key and a message (bytes), and returns
// the hex-encoded signature.
func Sign(privKeyHex string, message []byte) (string, error) {
	privBytes, err := hex.DecodeString(privKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid private key encoding: %w", err)
	}
	if len(privBytes) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("invalid private key length")
	}

	sig := ed25519.Sign(ed25519.PrivateKey(privBytes), message)
	return hex.EncodeToString(sig), nil
}

// Verify checks if a hex-encoded signature is valid for a given message
// using the hex-encoded public key.
func Verify(pubKeyHex string, message []byte, sigHex string) bool {
	pubBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return false
	}
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil || len(sigBytes) != ed25519.SignatureSize {
		return false
	}

	return ed25519.Verify(ed25519.PublicKey(pubBytes), message, sigBytes)
}
