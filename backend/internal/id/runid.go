package id

import (
	"crypto/rand"
	"errors"
	"regexp"
)

const RunIDLen = 16

var runIDPattern = regexp.MustCompile(`^[0-9A-Za-z]{16}$`)

// ValidateRunID returns true if s is exactly 16 alphanumeric ASCII characters.
func ValidateRunID(s string) bool {
	return runIDPattern.MatchString(s)
}

// GenerateRunID returns a cryptographically random 16-character alphanumeric ID.
func GenerateRunID() (string, error) {
	b := make([]byte, RunIDLen)
	n, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	if n != RunIDLen {
		return "", errors.New("unexpected short read from crypto/rand")
	}
	out := make([]byte, RunIDLen)
	for i := 0; i < RunIDLen; i++ {
		out[i] = assetAlphabet[int(b[i])%len(assetAlphabet)]
	}
	return string(out), nil
}
