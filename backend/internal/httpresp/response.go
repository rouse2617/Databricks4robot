package httpresp

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorBody is the standardized error envelope returned by all APIs.
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
	// Server-side faults (5xx) must leave a trace: without this, every 500 was
	// silent server-side and the actual cause (deploy failure, projection, argo,
	// etc.) was undiagnosable from logs — only the client saw the message.
	// 4xx are expected client errors and are intentionally not logged as errors
	// to avoid noise. (CYB-3569)
	if status >= http.StatusInternalServerError {
		method, path := "", ""
		if c != nil && c.Request != nil {
			method = c.Request.Method
			path = c.FullPath()
			if path == "" {
				path = c.Request.URL.Path
			}
		}
		slog.Error("http error response",
			"status", status,
			"code", code,
			"message", message,
			"method", method,
			"path", path,
			"request_id", requestID(c),
		)
	}
	c.JSON(status, ErrorBody{
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

func Unprocessable(c *gin.Context, code, message string, details map[string]any) {
	Error(c, http.StatusUnprocessableEntity, code, message, details)
}

func Internal(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, CodeInternalError, message, nil)
}
