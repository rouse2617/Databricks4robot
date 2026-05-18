package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/id"
)

// RequirePathAssetID validates :id as an 8-character alphanumeric asset id.
func RequirePathAssetID(c *gin.Context) (string, bool) {
	raw := strings.TrimSpace(c.Param("id"))
	if raw == "" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "asset id is required", nil)
		return "", false
	}
	if !id.ValidateAssetID(raw) {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid asset id: must be 8 alphanumeric characters", nil)
		return "", false
	}
	return raw, true
}
