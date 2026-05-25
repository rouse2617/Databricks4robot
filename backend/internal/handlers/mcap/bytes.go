package mcap

import (
	"cloud.google.com/go/storage"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
)

// Bytes proxies MCAP bytes from GCS (gs://) with HTTP Range support.
//
// Route: GET /api/v1/mcap-files/:id/bytes
//
// Behavior:
// - Always sets Accept-Ranges: bytes
// - No Range header: 200 with full content (Content-Length = total size)
// - Valid Range: 206 with Content-Range + correct Content-Length
// - Unsatisfiable/invalid Range: 416 with Content-Range: bytes */<size>
func (h *Handler) Bytes(c *gin.Context) {
	if h.repo == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable, "mcap repository not configured", nil)
		return
	}
	if h.bytesSrc == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable, "mcap bytes source not configured", nil)
		return
	}

	f, err := h.repo.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	if f == nil || f.GCSPath == "" {
		httpresp.NotFound(c, "MCAP_FILE_NOT_FOUND", "mcap file not found")
		return
	}
	if !strings.HasPrefix(f.GCSPath, "gs://") {
		httpresp.Error(c, http.StatusBadRequest, httpresp.CodeInvalidArgument, "mcap file gcs_path must be a gs:// URI", map[string]any{"gcs_path": f.GCSPath})
		return
	}

	size, contentType, err := h.bytesSrc.Stat(c.Request.Context(), f.GCSPath)
	if err != nil {
		if !writeBytesError(c, err) {
			httpresp.Internal(c, fmt.Sprintf("gcs stat failed: %v", err))
			return
		}
		return
		return
	}
	if size < 0 {
		httpresp.Internal(c, "invalid object size")
		return
	}

	c.Header("Accept-Ranges", "bytes")
	if contentType != "" {
		c.Header("Content-Type", contentType)
	} else {
		c.Header("Content-Type", "application/octet-stream")
	}

	rangeHdr := c.GetHeader("Range")
	if rangeHdr == "" {
		c.Header("Content-Length", strconv.FormatInt(size, 10))
		c.Status(http.StatusOK)
		if c.Request.Method == http.MethodHead {
			return
		}

		rc, err := h.bytesSrc.OpenFull(c.Request.Context(), f.GCSPath)
		if err != nil {
			if !writeBytesError(c, err) {
				httpresp.Internal(c, fmt.Sprintf("gcs read failed: %v", err))
				return
			}
			return
			return
		}
		defer func() { _ = rc.Close() }()

		_, _ = io.Copy(c.Writer, rc)
		return
	}

	start, endInclusive, ok := parseSingleByteRange(rangeHdr, size)
	if !ok {
		c.Header("Content-Range", fmt.Sprintf("bytes */%d", size))
		httpresp.Error(
			c,
			http.StatusRequestedRangeNotSatisfiable,
			httpresp.CodeInvalidRange,
			"invalid Range header",
			map[string]any{
				"range": rangeHdr,
				"size":  size,
			},
		)
		return
	}

	length := (endInclusive - start) + 1
	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, endInclusive, size))
	c.Header("Content-Length", strconv.FormatInt(length, 10))
	c.Status(http.StatusPartialContent)
	if c.Request.Method == http.MethodHead {
		return
	}

	rc, err := h.bytesSrc.OpenRange(c.Request.Context(), f.GCSPath, start, length)
	if err != nil {
		if !writeBytesError(c, err) {
			httpresp.Internal(c, fmt.Sprintf("gcs range read failed: %v", err))
			return
		}
		return
		return
	}
	defer func() { _ = rc.Close() }()

	_, _ = io.Copy(c.Writer, rc)
}

// parseSingleByteRange parses a single "bytes=..." Range header and returns an inclusive end.
// Supported forms:
// - bytes=<start>-<end>
// - bytes=<start>-
// - bytes=-<suffixLen>
func parseSingleByteRange(hdr string, size int64) (start, endInclusive int64, ok bool) {
	hdr = strings.TrimSpace(hdr)
	if !strings.HasPrefix(hdr, "bytes=") {
		return 0, 0, false
	}
	spec := strings.TrimSpace(strings.TrimPrefix(hdr, "bytes="))
	// Single range only.
	if strings.Contains(spec, ",") {
		return 0, 0, false
	}
	dash := strings.IndexByte(spec, '-')
	if dash < 0 {
		return 0, 0, false
	}
	a := strings.TrimSpace(spec[:dash])
	b := strings.TrimSpace(spec[dash+1:])
	if size <= 0 {
		return 0, 0, false
	}

	// Suffix: "-N" => last N bytes.
	if a == "" {
		if b == "" {
			return 0, 0, false
		}
		n, err := parseNonNegativeInt64(b)
		if err != nil || n <= 0 {
			return 0, 0, false
		}
		if n > size {
			n = size
		}
		start = size - n
		return start, size - 1, true
	}

	// Start must be non-negative.
	st, err := parseNonNegativeInt64(a)
	if err != nil {
		return 0, 0, false
	}
	if st >= size {
		return 0, 0, false
	}

	// Open ended: "start-"
	if b == "" {
		return st, size - 1, true
	}

	en, err := parseNonNegativeInt64(b)
	if err != nil {
		return 0, 0, false
	}
	if en < st {
		return 0, 0, false
	}
	if en >= size {
		en = size - 1
	}
	return st, en, true
}

func parseNonNegativeInt64(s string) (int64, error) {
	if s == "" {
		return 0, errors.New("empty")
	}
	// Reject leading "+" or "-" explicitly.
	if strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-") {
		return 0, errors.New("signed")
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil || v < 0 {
		return 0, errors.New("invalid")
	}
	return v, nil
}

// writeBytesError maps known bytes-source errors to HTTP error responses.
// It writes the response and returns true when the error is recognized.
// For unrecognized errors it returns false — the caller should respond 500.
func writeBytesError(c *gin.Context, err error) bool {
	if errors.Is(err, storage.ErrObjectNotExist) {
		httpresp.NotFound(c, httpresp.CodeMcapFileNotFound, "mcap file bytes not found")
		return true
	}
	return false
}
