import { apiClient } from "./client";

export interface MeResponse {
	authenticated: boolean;
	email?: string;
	role?: string;
}

export interface EmailLoginResponse {
	authenticated: boolean;
	email: string;
	role: string;
}

export const authApi = {
	emailLogin: (email: string) =>
		apiClient.post("/auth/email-login", { email }).then((r) => r.data as EmailLoginResponse),
	login: (token: string) =>
		apiClient.post("/auth/login", { token }).then((r) => r.data),
	me: () => apiClient.get("/auth/me").then((r) => r.data as MeResponse),
	logout: () => apiClient.post("/auth/logout").then((r) => r.data),
};
