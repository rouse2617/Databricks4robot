package workflow

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
)

func parseRFC3339QueryParam(c *gin.Context, name string) (time.Time, bool, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return time.Time{}, false, true
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid "+name+": must be RFC3339", map[string]any{
			"error": err.Error(),
			"name":  name,
		})
		return time.Time{}, false, false
	}
	return parsed, true, true
}
