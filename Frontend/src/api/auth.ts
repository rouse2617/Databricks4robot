import { apiClient } from "./client";

export const authApi = {
	login: (token: string) =>
		apiClient.post("/auth/login", { token }).then((r) => r.data),
	me: () => apiClient.get("/auth/me").then((r) => r.data),
	logout: () => apiClient.post("/auth/logout").then((r) => r.data),
};
