package httpresp

// Centralized error code constants used across all handlers and middleware.
const (
	CodeInvalidArgument        = "INVALID_ARGUMENT"
	CodeInvalidFilter          = "INVALID_FILTER"
	CodeInvalidState           = "INVALID_STATE"
	CodeInvalidTag             = "INVALID_TAG"
	CodeAssetNotFound          = "ASSET_NOT_FOUND"
	CodeDuplicateAssetID       = "DUPLICATE_ASSET_ID"
	CodeInvalidAlgoKey         = "INVALID_ALGO_KEY"
	CodeAlgoAlreadyRunning     = "ALGO_ALREADY_RUNNING"
	CodeInvalidStateTransition = "INVALID_STATE_TRANSITION"
	CodeConcurrentConflict     = "CONCURRENT_CONFLICT"
	CodeMissingRequiredField   = "MISSING_REQUIRED_FIELD"
	CodeMissingReason          = "MISSING_REASON"
	CodeInternalError          = "INTERNAL_ERROR"
	CodeUnauthorized           = "UNAUTHORIZED"
	CodeURITooLong             = "URI_TOO_LONG"
	CodeMissingIdempotencyKey  = "MISSING_IDEMPOTENCY_KEY"
	CodeIdempotencyConflict    = "IDEMPOTENCY_CONFLICT"
	CodeDeliveryNotFound       = "DELIVERY_NOT_FOUND"
	CodeRateLimited            = "RATE_LIMITED"
	CodeServiceUnavailable     = "SERVICE_UNAVAILABLE"
	CodeUnsupportedField       = "UNSUPPORTED_FIELD"
	CodeUnsupportedOperator    = "UNSUPPORTED_OPERATOR"
	CodeUnplannableQuery       = "UNPLANNABLE_QUERY"
)
