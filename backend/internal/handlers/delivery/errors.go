package delivery

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// writeDeliveryError maps known delivery errors to HTTP error responses.
// It writes the response and returns true when the error is recognized.
// For unrecognized errors it returns false — the caller should respond 500.
func writeDeliveryError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, repository.ErrOptimisticLock):
		httpresp.Conflict(c, httpresp.CodeConcurrentConflict,
			"delivery was modified concurrently", nil)

	default:
		return false
	}
	return true
}
