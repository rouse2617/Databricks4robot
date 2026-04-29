import axios from "axios";

export const TOKEN_KEY = "grace_token";

/** Default timeout for most API calls */
const DEFAULT_TIMEOUT = 30_000;

/** Longer timeout for lakehouse/analytics queries */
export const LAKEHOUSE_TIMEOUT = 90_000;

export const apiClient = axios.create({
  baseURL: "/api/v1",
  timeout: DEFAULT_TIMEOUT,
});

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY);
  if (token) {
    config.headers["X-Grace-Token"] = token;
  }
  return config;
});

apiClient.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem(TOKEN_KEY);
      // Preserve the current path so login page can redirect back
      const from = encodeURIComponent(window.location.pathname + window.location.search);
      window.location.href = `/login?from=${from}`;
    }
    return Promise.reject(err);
  }
);
