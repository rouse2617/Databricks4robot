# Behavior Spec — CYB-1135

## Delivery Cancel Smoke

### Cancel delivered delivery → 200
- **Given** a committed (delivered) delivery exists
- **When** `POST /api/v1/deliveries/{id}/cancel` with valid `cancelled_by` and `cancel_reason`
- **Then** response is HTTP 200 with `status: "cancelled"`

### Cancel non-existent delivery → 404
- **Given** no delivery with the given UUID exists
- **When** `POST /api/v1/deliveries/{id}/cancel`
- **Then** response is HTTP 404 with code `DELIVERY_NOT_FOUND`

## Delivery Retry Smoke

### Retry cancelled delivery → 201
- **Given** a cancelled delivery exists
- **When** `POST /api/v1/deliveries/{id}/retry`
- **Then** response is HTTP 201 with `status: "pending"` and a new `delivery_id`

## Delivery Ack Error Path

### Ack pending delivery → 422
- **Given** a pending delivery exists (e.g., the retry result above)
- **When** `POST /api/v1/deliveries/{id}/ack`
- **Then** response is HTTP 422 with code `INVALID_STATE` (only delivered can be acked)
