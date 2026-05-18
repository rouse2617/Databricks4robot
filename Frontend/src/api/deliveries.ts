import { apiClient } from "./client";
import type { Delivery, DeliveryItem, PaginatedResponse } from "./types";

export interface ListDeliveriesParams {
	page?: number;
	page_size?: number;
	status?: string;
}

export interface CreateDeliveryPayload {
	customer_id: string;
	contract_id?: string;
	note?: string;
	owner?: string;
	asset_ids?: string[];
}

export const deliveriesApi = {
	list: (params?: ListDeliveriesParams) => {
		const sp = new URLSearchParams();
		if (params?.page) sp.set("page", String(params.page));
		if (params?.page_size) sp.set("page_size", String(params.page_size));
		if (params?.status) sp.set("status", params.status);
		return apiClient
			.get<PaginatedResponse<Delivery>>(`/deliveries?${sp.toString()}`)
			.then((r) => r.data);
	},

	get: (id: string) =>
		apiClient.get<Delivery>(`/deliveries/${id}`).then((r) => r.data),

	commit: (payload: CreateDeliveryPayload, idempotencyKey: string) =>
		apiClient
			.post<Delivery>("/deliveries", payload, {
				headers: { "Idempotency-Key": idempotencyKey },
			})
			.then((r) => r.data),

	listItems: (deliveryId: string) =>
		apiClient
			.get<{ items: DeliveryItem[] }>(`/deliveries/${deliveryId}/items`)
			.then((r) => r.data.items),
};
