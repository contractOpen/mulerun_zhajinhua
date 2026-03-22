package handler

import (
	"regexp"
	"strings"
)

// WalletChain supported blockchain types
type WalletChain string

const (
	ChainEVM WalletChain = "evm"
	ChainTON WalletChain = "ton"
	ChainSOL WalletChain = "sol"
)

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
