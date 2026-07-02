package backfill

import "errors"

// MaxBackfillAssetCount caps assets per batch job to protect API/DB from overload.
const MaxBackfillAssetCount = 10_000

// MaxBackfillResultPayloadBytes caps JSON result payload size for upload API.
const MaxBackfillResultPayloadBytes = 2 * 1024 * 1024

var ErrTooManyAssets = errors.New("too many assets for one batch job")
