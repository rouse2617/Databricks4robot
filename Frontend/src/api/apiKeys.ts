import { apiClient } from "./client";

export interface ApiKey {
	id: string;
	keyPrefix: string;
	name: string;
	owner: string;
	scopes: string[];
	status: "active" | "revoked";
	expiresAt?: string | null;
	lastUsedAt?: string | null;
	createdBy: string;
	createdAt: string;
}

export interface CreateApiKeyRequest {
	name: string;
	owner?: string;
	scopes: string[];
	expiresAt?: string | null;
}

// CreateApiKeyResponse.key is the plaintext, returned ONLY once at creation.
export interface CreateApiKeyResponse {
	id: string;
	key: string;
	keyPrefix: string;
	name: string;
	owner: string;
	scopes: string[];
	expiresAt?: string | null;
	note: string;
}

/** Scopes a key can be granted. Keep in sync with backend RequireScope usage. */
export const AVAILABLE_SCOPES = [
	"assets:read",
	"assets:write",
	"pipeline:run",
] as const;

export const apiKeysApi = {
	list: () =>
		apiClient
			.get("/admin/api-keys")
			.then((r) => (r.data?.items ?? []) as ApiKey[]),
	create: (req: CreateApiKeyRequest) =>
		apiClient
			.post("/admin/api-keys", req)
			.then((r) => r.data as CreateApiKeyResponse),
	revoke: (id: string) =>
		apiClient.delete(`/admin/api-keys/${id}`).then((r) => r.data),
};
