package id

import (
	"crypto/rand"
	"errors"
	"regexp"
)

const AssetIDLen = 8

var assetIDPattern = regexp.MustCompile(`^[0-9A-Za-z]{8}$`)

const assetAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// ValidateAssetID returns true if s is exactly 8 alphanumeric characters (ASCII).
func ValidateAssetID(s string) bool {
	return assetIDPattern.MatchString(s)
}

// GenerateAssetID returns a cryptographically random 8-character alphanumeric ID.
func GenerateAssetID() (string, error) {
	b := make([]byte, AssetIDLen)
	n, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	if n != AssetIDLen {
		return "", errors.New("unexpected short read from crypto/rand")
	}
	out := make([]byte, AssetIDLen)
	for i := 0; i < AssetIDLen; i++ {
		out[i] = assetAlphabet[int(b[i])%len(assetAlphabet)]
	}
	return string(out), nil
}

// McapFileID uses the same alphabet and length as AssetID (8 alphanumeric ASCII).
// ValidateMcapFileID checks the same pattern as ValidateAssetID.
func ValidateMcapFileID(s string) bool {
	return ValidateAssetID(s)
}

// GenerateMcapFileID returns a cryptographically random 8-character alphanumeric ID.
func GenerateMcapFileID() (string, error) {
	return GenerateAssetID()
}

// ActionID uses the same alphabet and length as AssetID (8 alphanumeric ASCII).
// ValidateActionID checks the same pattern as ValidateAssetID.
func ValidateActionID(s string) bool {
	return ValidateAssetID(s)
}

// GenerateActionID returns a cryptographically random 8-character alphanumeric ID.
func GenerateActionID() (string, error) {
	return GenerateAssetID()
}
