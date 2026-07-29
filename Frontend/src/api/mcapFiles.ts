import { apiClient } from "./client";
import type { McapFile, PaginatedResponse } from "./types";

export interface ListMcapFilesParams {
	page?: number;
	page_size?: number;
	ingest_state?: string;
	owner?: string;
	// CYB-4445: exact-match filter; backend enforces 8-char alphanumeric.
	mcap_file_id?: string;
}

export const mcapFilesApi = {
	list: (params?: ListMcapFilesParams) => {
		const qp = new URLSearchParams();
		if (params?.page) qp.append("page", String(params.page));
		if (params?.page_size) qp.append("page_size", String(params.page_size));
		if (params?.ingest_state) qp.append("ingest_state", params.ingest_state);
		if (params?.owner) qp.append("owner", params.owner);
		if (params?.mcap_file_id) qp.append("mcap_file_id", params.mcap_file_id);
		return apiClient
			.get<PaginatedResponse<McapFile>>("/mcap-files", { params: qp })
			.then((r) => r.data);
	},

	get: (id: string) =>
		apiClient.get<McapFile>(`/mcap-files/${id}`).then((r) => r.data),

	finalize: (id: string) =>
		apiClient
			.post(`/mcap-files/${encodeURIComponent(id)}/finalize`)
			.then((r) => r.data),
};
