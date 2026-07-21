package httpresp

// Centralized error code constants used across all handlers and middleware.
const (
	CodeInvalidArgument          = "INVALID_ARGUMENT"
	CodeInvalidRange             = "INVALID_RANGE"
	CodeInvalidFilter            = "INVALID_FILTER"
	CodeInvalidState             = "INVALID_STATE"
	CodeInvalidTag               = "INVALID_TAG"
	CodeTagSourceInvalid         = "TAG_SOURCE_INVALID"
	CodeTagImmutable             = "TAG_IMMUTABLE"
	CodeAssetNotFound            = "ASSET_NOT_FOUND"
	CodeConfigNotFound           = "CONFIG_NOT_FOUND"
	CodeAssetNotPreviewable      = "ASSET_NOT_PREVIEWABLE"
	CodeMcapFileNotFound         = "MCAP_FILE_NOT_FOUND"
	CodeDuplicateAssetID         = "DUPLICATE_ASSET_ID"
	CodeDuplicateMcapFileID      = "DUPLICATE_MCAP_FILE_ID"
	CodeDuplicateHash            = "DUPLICATE_HASH"
	CodeAlgoRunNotFound          = "ALGO_RUN_NOT_FOUND"
	CodeInvalidAlgoKey           = "INVALID_ALGO_KEY"
	CodeAlgoAlreadyRunning       = "ALGO_ALREADY_RUNNING"
	CodeInvalidStateTransition   = "INVALID_STATE_TRANSITION"
	CodeConcurrentConflict       = "CONCURRENT_CONFLICT"
	CodeMissingRequiredField     = "MISSING_REQUIRED_FIELD"
	CodeMissingReason            = "MISSING_REASON"
	CodeInternalError            = "INTERNAL_ERROR"
	CodeUnauthorized             = "UNAUTHORIZED"
	CodeURITooLong               = "URI_TOO_LONG"
	CodeMissingIdempotencyKey    = "MISSING_IDEMPOTENCY_KEY"
	CodeIdempotencyConflict      = "IDEMPOTENCY_CONFLICT"
	CodeDeliveryNotFound         = "DELIVERY_NOT_FOUND"
	CodeDeliveryRuleFailed       = "DELIVERY_RULE_FAILED"
	CodeRateLimited              = "RATE_LIMITED"
	CodeServiceUnavailable       = "SERVICE_UNAVAILABLE"
	CodeUnsupportedField         = "UNSUPPORTED_FIELD"
	CodeUnsupportedOperator      = "UNSUPPORTED_OPERATOR"
	CodeCustomerNotFound         = "CUSTOMER_NOT_FOUND"          // CYB-1070
	CodeHierarchyViolation       = "ASSET_HIERARCHY_VIOLATION"   // CYB-1164
	CodeTagKeyExists             = "TAG_KEY_EXISTS"              // CYB-3246
	CodeTagNotFound              = "TAG_NOT_FOUND"               // CYB-3246
	CodeSubscriptionTaskNotFound = "SUBSCRIPTION_TASK_NOT_FOUND" // CYB-3778
)
