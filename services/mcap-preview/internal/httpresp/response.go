// Package httpresp mirrors the cyber-databrew error envelope shape so that
// preview clients see a consistent {code, message, request_id, details} body.
//
// Kept intentionally tiny — this service has very few error paths.
package httpresp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeInvalidArgument     = "INVALID_ARGUMENT"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeAssetNotFound       = "ASSET_NOT_FOUND"
	CodeMcapFileNotFound    = "MCAP_FILE_NOT_FOUND"
	CodeAssetNotPreviewable = "ASSET_NOT_PREVIEWABLE"
	CodeUpstreamError       = "UPSTREAM_ERROR"
	CodeInternalError       = "INTERNAL_ERROR"
	CodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
)

// ErrorBody is the standardized error envelope returned by all endpoints.
type ErrorBody struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	RequestID string         `json:"request_id"`
	Details   map[string]any `json:"details,omitempty"`
}

func requestID(c *gin.Context) string {
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return c.GetHeader("X-Request-ID")
}

func Error(c *gin.Context, status int, code, message string, details map[string]any) {
	c.AbortWithStatusJSON(status, ErrorBody{
		Code:      code,
		Message:   message,
		RequestID: requestID(c),
		Details:   details,
	})
}

func BadRequest(c *gin.Context, code, message string, details map[string]any) {
	Error(c, http.StatusBadRequest, code, message, details)
}

func Unauthorized(c *gin.Context, code, message string) {
	Error(c, http.StatusUnauthorized, code, message, nil)
}

func NotFound(c *gin.Context, code, message string) {
	Error(c, http.StatusNotFound, code, message, nil)
}

func Conflict(c *gin.Context, code, message string, details map[string]any) {
	Error(c, http.StatusConflict, code, message, details)
}

func Unavailable(c *gin.Context, code, message string) {
	Error(c, http.StatusServiceUnavailable, code, message, nil)
}

func Internal(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, CodeInternalError, message, nil)
}
