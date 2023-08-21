package nip44

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"golang.org/x/crypto/chacha20"
)

// ComputeSharedSecret returns a shared secret key used to encrypt messages.
// The private and public keys should be hex encoded.
// Uses the Diffie-Hellman key exchange (ECDH) (RFC 4753).
func ComputeSharedSecret(pub string, sk string) ([]byte, error) {
	privKeyBytes, err := hex.DecodeString(sk)
	if err != nil {
		return nil, fmt.Errorf("error decoding sender private key: %w", err)
	}
	privKey, _ := btcec.PrivKeyFromBytes(privKeyBytes)

	// adding 02 to signal that this is a compressed public key (33 bytes)
	pubKeyBytes, err := hex.DecodeString("02" + pub)
	if err != nil {
		return nil, fmt.Errorf("error decoding hex string of receiver public key '%s': %w", "02"+pub, err)
	}
	pubKey, err := btcec.ParsePubKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing receiver public key '%s': %w", "02"+pub, err)
	}

	sharedSecret := btcec.GenerateSharedSecret(privKey, pubKey)
	hash := sha256.Sum256(sharedSecret)
	return hash[:], nil
}

// Encrypt encrypts message with key using xchacha20.
// key should be the shared secret generated by ComputeSharedSecret.
// Returns: base64(1 + nonce + encrypted_bytes).
func Encrypt(message string, key []byte) (string, error) {
	nonce := make([]byte, 24)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("error creating nonce: %w", err)
	}

	return encryptWithNonce(message, key, nonce)
}

func encryptWithNonce(message string, key []byte, nonce []byte) (string, error) {
	cipher, err := chacha20.NewUnauthenticatedCipher(key, nonce)
	if err != nil {
		return "", fmt.Errorf("error creating cipher: %w", err)
	}

	msg := []byte(message)
	cipher.XORKeyStream(msg, msg)

	payload := make([]byte, 1+24+len(msg))
	payload[0] = 1
	copy(payload[1:25], nonce)
	copy(payload[25:], msg)

	return base64.StdEncoding.EncodeToString(payload), nil
}

// Decrypt decrypts a content string using the shared secret key.
// The inverse operation to message -> Encrypt(message, key).
func Decrypt(payload string, key []byte) (string, error) {
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("invalid base64: %w", err)
	}

	if data[0] != 1 {
		return "", fmt.Errorf("unknown version: %d", data[0])
	}
	if len(data) <= 25 {
		return "", fmt.Errorf("invalid payload, too small: %d", len(data))
	}

	nonce := data[1:25]
	msg := data[25:]
	cipher, err := chacha20.NewUnauthenticatedCipher(key, nonce)
	if err != nil {
		return "", fmt.Errorf("error creating cipher: %w", err)
	}

	cipher.XORKeyStream(msg, msg)

	return string(msg), nil
}
