import axios from "axios";

export const UNAUTHORIZED_EVENT = "***";
const DEV_ACCESS_TOKEN = import.meta.env.DEV
	? (import.meta.env.VITE_DEV_ACCESS_TOKEN ?? "").trim()
	: "";

/** Default timeout for most API calls */
const DEFAULT_TIMEOUT = 30_000;

/** Longer timeout for lakehouse/analytics queries */
export const LAKEHOUSE_TIMEOUT = 90_000;

export const apiClient = axios.create({
	baseURL: "/api/v1",
	timeout: DEFAULT_TIMEOUT,
});

apiClient.interceptors.request.use((config) => {
	if (DEV_ACCESS_TOKEN) {
		config.headers = config.headers ?? {};
		config.headers["X-Databrew-Token"] = DEV_ACCESS_TOKEN;
	}
	config.withCredentials = true;
	return config;
});

apiClient.interceptors.response.use(
	(res) => res,
	(err) => {
		if (err.response?.status === 401) {
			window.dispatchEvent(new Event(UNAUTHORIZED_EVENT));
		}
		return Promise.reject(err);
	},
);
