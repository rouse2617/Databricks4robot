// Centralized helper for extracting a user-facing error message from
// any thrown error: Axios HTTP errors, Axios network errors, or plain Errors.
//
// Without this helper, callers tend to write `err: any` + chained optional
// access (`err?.response?.data?.message ?? "操作失败"`), which silently swallows
// network-layer errors and produces confusing UX when the request never
// reached the server.

import { AxiosError } from "axios";

interface BackendErrorEnvelope {
  message?: string;
  code?: string;
}

function isBackendErrorEnvelope(value: unknown): value is BackendErrorEnvelope {
  return typeof value === "object" && value !== null && "message" in value;
}

// Returns a non-empty user-facing error message. `fallback` is used when the
// error has no extractable detail (e.g. a thrown non-Error value).
export function extractApiErrorMessage(err: unknown, fallback = "操作失败"): string {
  if (err instanceof AxiosError) {
    const data = err.response?.data;
    if (isBackendErrorEnvelope(data) && data.message) {
      return data.message;
    }
    if (err.response) {
      return `${err.response.status} ${err.response.statusText || "请求失败"}`;
    }
    if (err.code === "ERR_NETWORK" || err.message === "Network Error") {
      return "网络连接失败，请检查后端服务";
    }
    if (err.code === "ECONNABORTED") {
      return "请求超时，请稍后重试";
    }
    return err.message || fallback;
  }
  if (err instanceof Error) {
    return err.message || fallback;
  }
  return fallback;
}
