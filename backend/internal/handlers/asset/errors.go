package asset

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	assetUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/asset"
)

// mapAssetError maps known asset usecase errors to HTTP responses.
// Returns true if the error was recognised (response already written),
// false if the caller should fall back to a 500 Internal.
func mapAssetError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, assetUC.ErrNotFound):
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
	case errors.Is(err, assetUC.ErrInvalidTag):
		httpresp.Unprocessable(c, httpresp.CodeInvalidTag, err.Error(), nil)
	case errors.Is(err, assetUC.ErrTagSourceInvalid):
		httpresp.Unprocessable(c, httpresp.CodeTagSourceInvalid, err.Error(), nil)
	case errors.Is(err, assetUC.ErrTagImmutable):
		httpresp.Conflict(c, httpresp.CodeTagImmutable, err.Error(), nil)
	case errors.Is(err, assetUC.ErrCustomerNotFound):
		httpresp.Unprocessable(c, httpresp.CodeCustomerNotFound, err.Error(), nil)
	case errors.Is(err, assetUC.ErrInvalidActionLabel):
		// CYB-3268: keep the legacy INVALID_ACTION code so clients that already
		// handle the old actions API see the same error envelope.
		httpresp.Unprocessable(c, "INVALID_ACTION", err.Error(), nil)
	case errors.Is(err, assetUC.ErrMcapFileIDRequired),
		errors.Is(err, assetUC.ErrInvalidRange),
		errors.Is(err, assetUC.ErrDurationTooSmall):
		httpresp.Unprocessable(c, httpresp.CodeInvalidState, err.Error(), nil)
	case errors.Is(err, assetUC.ErrLogicalAssetNotFound),
		errors.Is(err, assetUC.ErrLogicalAssetTypeMismatch):
		httpresp.Unprocessable(c, httpresp.CodeInvalidState, err.Error(), nil)
	case errors.Is(err, assetUC.ErrInvalidMcapFileID),
		errors.Is(err, assetUC.ErrMcapFileNotFound):
		httpresp.Unprocessable(c, httpresp.CodeInvalidArgument, err.Error(), nil)
	case errors.Is(err, assetUC.ErrInvalidAssetID):
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
	case errors.Is(err, assetUC.ErrAssetIDTaken):
		httpresp.Conflict(c, httpresp.CodeDuplicateAssetID, err.Error(), nil)
	case errors.Is(err, repository.ErrOptimisticLock):
		httpresp.Conflict(c, httpresp.CodeConcurrentConflict,
			"asset was modified concurrently; reload and retry", nil)
	default:
		return false
	}
	return true
}
