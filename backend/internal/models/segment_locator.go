package models

import (
	"crypto/sha1"
	"encoding/hex"
	"strconv"
)

// ComputeSegmentLocator returns the 40-char hex SHA-1 digest of
// mcapFileID + strconv.FormatInt(startNs, 10) + strconv.FormatInt(endNs, 10).
// This is a pure, deterministic function used to uniquely identify a segment
// by its source file and time range.
func ComputeSegmentLocator(mcapFileID string, startNs, endNs int64) string {
	h := sha1.New()
	h.Write([]byte(mcapFileID))
	h.Write([]byte(strconv.FormatInt(startNs, 10)))
	h.Write([]byte(strconv.FormatInt(endNs, 10)))
	return hex.EncodeToString(h.Sum(nil))
}
