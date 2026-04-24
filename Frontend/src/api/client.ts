import axios from "axios";

const TOKEN_KEY = "grace_token";

export const apiClient = axios.create({
  baseURL: "/api/v1",
  timeout: 30_000,
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
      window.location.href = "/login";
    }
    return Promise.reject(err);
  }
);
