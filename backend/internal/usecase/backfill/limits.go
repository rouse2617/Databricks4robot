package backfill

import "errors"

// MaxBackfillAssetCount caps assets per batch job to protect API/DB from overload.
const MaxBackfillAssetCount = 10_000

var ErrTooManyAssets = errors.New("too many assets for one batch job")
