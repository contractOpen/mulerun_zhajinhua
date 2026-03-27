package handler

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestValidateWalletAddress_EVM(t *testing.T) {
	tests := []struct {
		address string
		valid   bool
	}{
		{"0x742d35Cc6634C0532925a3b844Bc9e7595f2bD68", true},
		{"0x0000000000000000000000000000000000000000", true},
		{"0xABCDEF1234567890abcdef1234567890ABCDEF12", true},
		{"", false},
		{"0x123", false},                           // too short
		{"742d35Cc6634C0532925a3b844Bc9e7595f2bD68", false}, // missing 0x
		{"0xGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG", false}, // invalid hex
		{"0x742d35Cc6634C0532925a3b844Bc9e7595f2bD6", false},  // 39 chars
	}

	for _, tt := range tests {
		got := ValidateWalletAddress(tt.address, ChainEVM)
		if got != tt.valid {
			t.Errorf("ValidateWalletAddress(%q, EVM) = %v, want %v", tt.address, got, tt.valid)
		}
	}
}

func TestValidateWalletAddress_TON(t *testing.T) {
	tests := []struct {
		address string
		valid   bool
	}{
		// User-friendly format (48+ chars starting with EQ or UQ)
		{"EQBvW8Z5huBkMJYdnfAEM5JqTNkuWX3diqYENkWsIL0XggGG", true},
		{"UQBvW8Z5huBkMJYdnfAEM5JqTNkuWX3diqYENkWsIL0XggGG", true},
		// Raw format with 0: prefix (64 hex chars)
		{"0:abcdef1234567890abcdef1234567890abcdef1234567890abcdef12345678ab", true},
		// Raw hex 64 chars
		{"abcdef1234567890abcdef1234567890abcdef1234567890abcdef12345678ab", true},
		{"", false},
		{"EQshort", false},
	}

	for _, tt := range tests {
		got := ValidateWalletAddress(tt.address, ChainTON)
		if got != tt.valid {
			t.Errorf("ValidateWalletAddress(%q, TON) = %v, want %v", tt.address, got, tt.valid)
		}
	}
}

func TestValidateWalletAddress_SOL(t *testing.T) {
	tests := []struct {
		address string
		valid   bool
	}{
		{"7EcDhSYGxXyscszYEp35KHN8vvw3svAuLKTzXwCFLtV", true},
		{"11111111111111111111111111111111", true}, // 32 chars, valid base58
		{"", false},
		{"0x742d35Cc6634C0532925a3b844Bc9e7595f2bD68", false}, // EVM address
		{"short", false},
	}

	for _, tt := range tests {
		got := ValidateWalletAddress(tt.address, ChainSOL)
		if got != tt.valid {
			t.Errorf("ValidateWalletAddress(%q, SOL) = %v, want %v", tt.address, got, tt.valid)
		}
	}
}

func TestValidateWalletAddress_UnknownChain(t *testing.T) {
	got := ValidateWalletAddress("someaddress", WalletChain("btc"))
	if got {
		t.Error("unknown chain should return false")
	}
}

func TestValidateWalletAddress_Whitespace(t *testing.T) {
	// Should trim whitespace
	got := ValidateWalletAddress("  0x742d35Cc6634C0532925a3b844Bc9e7595f2bD68  ", ChainEVM)
	if !got {
		t.Error("should accept address with surrounding whitespace")
	}
}

func TestDetectChain_EVM(t *testing.T) {
	chain := DetectChain("0x742d35Cc6634C0532925a3b844Bc9e7595f2bD68")
	if chain != ChainEVM {
		t.Errorf("expected ChainEVM, got %q", chain)
	}
}

func TestDetectChain_TON_EQ(t *testing.T) {
	chain := DetectChain("EQBvW8Z5huBkMJYdnfAEM5JqTNkuWX3diqYENkWsIL0XggGG")
	if chain != ChainTON {
		t.Errorf("expected ChainTON, got %q", chain)
	}
}

func TestDetectChain_TON_UQ(t *testing.T) {
	chain := DetectChain("UQBvW8Z5huBkMJYdnfAEM5JqTNkuWX3diqYENkWsIL0XggGG")
	if chain != ChainTON {
		t.Errorf("expected ChainTON, got %q", chain)
	}
}

func TestDetectChain_TON_Raw(t *testing.T) {
	chain := DetectChain("0:abcdef1234567890abcdef1234567890abcdef1234567890abcdef12345678ab")
	if chain != ChainTON {
		t.Errorf("expected ChainTON, got %q", chain)
	}
}

func TestDetectChain_SOL(t *testing.T) {
	chain := DetectChain("7EcDhSYGxXyscszYEp35KHN8vvw3svAuLKTzXwCFLtV")
	if chain != ChainSOL {
		t.Errorf("expected ChainSOL, got %q", chain)
	}
}

func TestDetectChain_Unknown(t *testing.T) {
	chain := DetectChain("not-a-valid-address!!!")
	if chain != "" {
		t.Errorf("expected empty string for unknown, got %q", chain)
	}
}

func TestVerifyWalletSignature_EVM(t *testing.T) {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	address := crypto.PubkeyToAddress(privKey.PublicKey).Hex()
	nonce := "testnonce123"
	message := BuildAuthMessage(address, ChainEVM, nonce)
	hash := crypto.Keccak256(prefixedEVMMessage([]byte(message)))
	sig, err := crypto.Sign(hash, privKey)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	sig[64] += 27

	if err := VerifyWalletSignature(address, ChainEVM, message, "0x"+commonBytesToHex(sig)); err != nil {
		t.Fatalf("VerifyWalletSignature failed: %v", err)
	}
}

func TestVerifyWalletSignature_SOL(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	address := base58.Encode(pub)
	message := BuildAuthMessage(address, ChainSOL, "solnonce123")
	sig := ed25519.Sign(priv, []byte(message))
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	if err := VerifyWalletSignature(address, ChainSOL, message, sigB64); err != nil {
		t.Fatalf("VerifyWalletSignature failed: %v", err)
	}
}

func TestVerifyWalletSignature_InvalidSignature(t *testing.T) {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	address := crypto.PubkeyToAddress(privKey.PublicKey).Hex()
	message := BuildAuthMessage(address, ChainEVM, "nonce")
	hash := crypto.Keccak256(prefixedEVMMessage([]byte("tampered")))
	sig, err := crypto.Sign(hash, privKey)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	sig[64] += 27

	if err := VerifyWalletSignature(address, ChainEVM, message, "0x"+commonBytesToHex(sig)); err == nil {
		t.Fatal("expected invalid signature verification to fail")
	}
}

func commonBytesToHex(b []byte) string {
	const hextable = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hextable[v>>4]
		out[i*2+1] = hextable[v&0x0f]
	}
	return string(out)
}
