# Spec delta — auth

## ADDED

### Requirement: privileged API-key scopes must be gated by AuthMethod

`POST /api/v1/admin/api-keys` MUST reject requests whose `scopes` field contains any of the following **privileged** values when the caller authenticated via anything other than the static admin token (i.e. when `Principal.AuthMethod != AuthMethodStaticToken`):

- `*`
- `apikeys:manage`

#### Scenario: JWT admin session tries to mint a wildcard-scope key

- **Given** an admin JWT obtained via `POST /api/v1/auth/email-login` (`AuthMethod = "jwt"`)
- **When** the client sends `POST /api/v1/admin/api-keys` with `scopes: ["*"]`
- **Then** the server responds `403 FORBIDDEN` with error code `"FORBIDDEN"` and a message noting that the static admin token is required for the requested scope; the api_keys table is unchanged.

#### Scenario: JWT admin session tries to mint an `apikeys:manage` key

- **Given** an admin JWT (`AuthMethod = "jwt"`)
- **When** the client sends `POST /api/v1/admin/api-keys` with `scopes: ["apikeys:manage"]`
- **Then** the server responds `403 FORBIDDEN`; the api_keys table is unchanged.

#### Scenario: static admin token issues a wildcard-scope key (bootstrap path)

- **Given** the caller authenticated with the static admin token (`AuthMethod = "static-token"`)
- **When** the client sends `POST /api/v1/admin/api-keys` with `scopes: ["*"]`
- **Then** the server responds `201 CREATED` and returns the plaintext key exactly once; a new row is inserted with `scopes = ["*"]`.

#### Scenario: normal scopes pass through for any admin auth method

- **Given** an admin JWT (`AuthMethod = "jwt"`)
- **When** the client sends `POST /api/v1/admin/api-keys` with `scopes: ["assets:read", "assets:write"]`
- **Then** the server responds `201 CREATED` and inserts the key.

### Requirement: shared-secret tokens must be compared in constant time

Every server-side comparison of a caller-supplied shared secret against a configured secret (X-Databrew-Token / Authorization: Bearer for static token, X-Admin-Token for admin token, SDK legacy static token, Argo webhook token) MUST use `crypto/subtle.ConstantTimeCompare` on equal-length byte slices. Empty configured secrets MUST fail closed (never authenticate).

#### Scenario: empty configured token rejects any credential

- **Given** `DATABREW_TOKEN=""` (misconfiguration)
- **When** any client sends a request under `StaticTokenAuth` / `JWTAuth` / `AdminTokenAuth`
- **Then** the server responds `401 UNAUTHORIZED`; no request is ever admitted while the configured token is empty.

#### Scenario: correct token authenticates with or without Bearer prefix

- **Given** `DATABREW_TOKEN=abc123` and the client sends `Authorization: Bearer abc123` (or `X-Databrew-Token: abc123`, or plain `abc123`)
- **When** any of the above authentication middlewares run
- **Then** the request is admitted; comparison uses `subtle.ConstantTimeCompare` after stripping the optional `Bearer ` prefix.

#### Scenario: length-mismatched token rejects

- **Given** `DATABREW_TOKEN=abc123` and the client sends `abc12` (shorter)
- **When** middleware runs
- **Then** the request is rejected in constant time relative to the length check; no byte-boundary timing information leaks.

## MODIFIED

### Requirement: email-login admin elevation is a time-limited operator session

`POST /api/v1/auth/email-login` continues to mint a `role="admin"` JWT (24h TTL) for emails whose domain matches `ALLOWED_DOMAIN` and whose value is in the `ADMIN_EMAILS` list (`IsAdminEmail(email)`). This behavior is unchanged from the previous implementation, but is now explicitly documented as a **time-limited operator session**: the resulting JWT MUST NOT be usable to mint any credential that outlives the 24h session. Enforcement lives in the API-key scope guard above.

The proof-of-email-ownership gap (no magic link, no OIDC) is intentional for internal use and is acknowledged as a limitation; a follow-up ticket will evaluate stronger proof-of-possession if the endpoint's exposure profile changes.
