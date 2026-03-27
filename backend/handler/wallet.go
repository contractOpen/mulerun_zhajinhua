package handler

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// WalletChain supported blockchain types
type WalletChain string

const (
	ChainEVM WalletChain = "evm"
	ChainTON WalletChain = "ton"
	ChainSOL WalletChain = "sol"
)

const AuthChallengeTTLSeconds = 300

// ValidateWalletAddress checks if the given address is valid for the specified chain
func ValidateWalletAddress(address string, chain WalletChain) bool {
	address = strings.TrimSpace(address)
	if address == "" {
		return false
	}

	switch chain {
	case ChainEVM:
		// Ethereum: 0x followed by 40 hex characters
		return regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`).MatchString(address)
	case ChainTON:
		// TON: User-friendly format (48 chars base64) or raw format
		// User-friendly: starts with EQ or UQ, ~48 chars
		if len(address) >= 48 && (strings.HasPrefix(address, "EQ") || strings.HasPrefix(address, "UQ")) {
			return true
		}
		// Raw format: 0: prefix + 64 hex
		if regexp.MustCompile(`^0:[0-9a-fA-F]{64}$`).MatchString(address) {
			return true
		}
		// Also accept raw hex 64 chars
		return regexp.MustCompile(`^[0-9a-fA-F]{64}$`).MatchString(address)
	case ChainSOL:
		// Solana: Base58, 32-44 characters
		return regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`).MatchString(address)
	}
	return false
}

// DetectChain tries to detect the blockchain from the address format
func DetectChain(address string) WalletChain {
	address = strings.TrimSpace(address)
	if strings.HasPrefix(address, "0x") && len(address) == 42 {
		return ChainEVM
	}
	if strings.HasPrefix(address, "EQ") || strings.HasPrefix(address, "UQ") || strings.HasPrefix(address, "0:") {
		return ChainTON
	}
	if regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`).MatchString(address) {
		return ChainSOL
	}
	return ""
}

func GenerateAuthNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func BuildAuthMessage(address string, chain WalletChain, nonce string) string {
	return fmt.Sprintf(
		"ZhaJinHua Login\nWallet: %s\nChain: %s\nNonce: %s\n\nSign this message to authenticate.",
		strings.TrimSpace(address),
		strings.ToUpper(string(chain)),
		nonce,
	)
}

func VerifyWalletSignature(address string, chain WalletChain, message string, signature string) error {
	switch chain {
	case ChainEVM:
		return verifyEVMWalletSignature(address, message, signature)
	case ChainSOL:
		return verifySOLWalletSignature(address, message, signature)
	case ChainTON:
		return fmt.Errorf("TON signature verification not implemented")
	default:
		return fmt.Errorf("unsupported chain: %s", chain)
	}
}

func verifyEVMWalletSignature(address string, message string, signature string) error {
	sigHex := strings.TrimPrefix(strings.TrimSpace(signature), "0x")
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return fmt.Errorf("invalid hex signature")
	}
	if len(sig) != 65 {
		return fmt.Errorf("invalid EVM signature length")
	}
	if sig[64] >= 27 {
		sig[64] -= 27
	}
	if sig[64] > 1 {
		return fmt.Errorf("invalid EVM recovery id")
	}

	hash := crypto.Keccak256(prefixedEVMMessage([]byte(message)))
	pubKey, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return fmt.Errorf("failed to recover public key")
	}
	recovered := crypto.PubkeyToAddress(*pubKey)
	expected := common.HexToAddress(strings.TrimSpace(address))
	if !strings.EqualFold(recovered.Hex(), expected.Hex()) {
		return fmt.Errorf("signature address mismatch")
	}
	return nil
}

func prefixedEVMMessage(message []byte) []byte {
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(message))
	return append([]byte(prefix), message...)
}

func verifySOLWalletSignature(address string, message string, signature string) error {
	pubKey := base58.Decode(strings.TrimSpace(address))
	if len(pubKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid Solana public key")
	}

	sigBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature))
	if err != nil {
		return fmt.Errorf("invalid Solana signature encoding")
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("invalid Solana signature length")
	}
	if !ed25519.Verify(ed25519.PublicKey(pubKey), []byte(message), sigBytes) {
		return fmt.Errorf("invalid Solana signature")
	}
	return nil
}
