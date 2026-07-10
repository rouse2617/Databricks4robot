import { apiClient } from "./client";

export interface TagRegistryItem {
	key: string;
	description?: string;
	type: string;
	values?: string[];
	max_length?: number;
}

export interface TagRegistryResponse {
	items: TagRegistryItem[];
}

export const tagRegistryApi = {
	list: () =>
		apiClient
			.get<TagRegistryResponse>("/tag-registry")
			.then((r) => r.data.items),
};

// ─── Admin CRUD (CYB-3246 Phase 2) ───
// Managed tag definitions live in the DB and are editable without a restart.
// Guarded by AdminTokenOrAdminRole — a logged-in admin-role web session
// (ADMIN_EMAILS) is authorized via the session cookie; non-admins get 401.

export interface TagRegistryEntry {
	key: string;
	description?: string;
	type: "enum" | "string";
	values?: string[];
	max_length?: number;
	propagation?: "none" | "descendants";
	created_by?: string;
	created_at?: string;
	updated_at?: string;
}

export interface TagDefRequest {
	key: string;
	description?: string;
	type: "enum" | "string";
	values?: string[];
	max_length?: number;
	propagation?: "none" | "descendants";
}

// skipAuthRedirect: a 401 here means "not an admin", not "session expired" —
// do not tear down the web session for a logged-in non-admin.
const adminCfg = { skipAuthRedirect: true } as const;

export const tagRegistryAdminApi = {
	list: () =>
		apiClient
			.get<{ items: TagRegistryEntry[] }>("/admin/tag-registry", adminCfg)
			.then((r) => r.data.items),
	create: (body: TagDefRequest) =>
		apiClient
			.post<TagRegistryEntry>("/admin/tag-registry", body, adminCfg)
			.then((r) => r.data),
	update: (key: string, body: TagDefRequest) =>
		apiClient
			.patch<TagRegistryEntry>(
				`/admin/tag-registry/${encodeURIComponent(key)}`,
				body,
				adminCfg,
			)
			.then((r) => r.data),
	remove: (key: string) =>
		apiClient.delete(
			`/admin/tag-registry/${encodeURIComponent(key)}`,
			adminCfg,
		),
};
