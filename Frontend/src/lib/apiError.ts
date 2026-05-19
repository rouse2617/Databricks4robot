// Centralized helpers for surfacing backend error envelopes.
//
// Backend responses follow `{ code, message, request_id, details }`
// (see CLAUDE.md → "API Conventions"). UI code should prefer
// `describeApiError` for the structured shape, and only fall back to
// `extractApiErrorMessage` when a single string is enough.
//
// Without these helpers, callers tend to write `err: any` + chained optional
// access (`err?.response?.data?.message ?? "操作失败"`), which silently swallows
// network-layer errors and produces confusing UX when the request never
// reached the server.

import { AxiosError } from "axios";

export interface BackendErrorEnvelope {
	code?: string;
	message?: string;
	request_id?: string;
	details?: unknown;
}

export interface DescribedApiError {
	/** Human-facing message (always non-empty). */
	message: string;
	/** Backend error code, if provided. */
	code?: string;
	/** Backend `request_id` for support / log lookup. */
	requestId?: string;
	/** HTTP status code, when the request reached the server. */
	status?: number;
	/** True when the request never reached the server (network / timeout). */
	isNetwork: boolean;
}

function isBackendErrorEnvelope(value: unknown): value is BackendErrorEnvelope {
	return (
		typeof value === "object" &&
		value !== null &&
		("message" in value || "code" in value || "request_id" in value)
	);
}

/** Structured description of an API error for richer UI surfaces. */
export function describeApiError(
	err: unknown,
	fallback = "操作失败",
): DescribedApiError {
	if (err instanceof AxiosError) {
		const data = err.response?.data;
		const envelope = isBackendErrorEnvelope(data) ? data : undefined;

		if (err.response) {
			const message =
				envelope?.message ||
				`${err.response.status} ${err.response.statusText || "请求失败"}`;
			return {
				message,
				code: envelope?.code,
				requestId: envelope?.request_id,
				status: err.response.status,
				isNetwork: false,
			};
		}

		if (err.code === "ERR_NETWORK" || err.message === "Network Error") {
			return {
				message: "网络连接失败，请检查后端服务",
				isNetwork: true,
			};
		}
		if (err.code === "ECONNABORTED") {
			return {
				message: "请求超时，请稍后重试",
				isNetwork: true,
			};
		}
		return { message: err.message || fallback, isNetwork: true };
	}
	if (err instanceof Error) {
		return { message: err.message || fallback, isNetwork: false };
	}
	return { message: fallback, isNetwork: false };
}

// Returns a non-empty user-facing error message. `fallback` is used when the
// error has no extractable detail (e.g. a thrown non-Error value).
export function extractApiErrorMessage(
	err: unknown,
	fallback = "操作失败",
): string {
	return describeApiError(err, fallback).message;
}
