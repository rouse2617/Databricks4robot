// @vitest-environment jsdom

import {
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import SettingsPage from "./SettingsPage";

function clearLocalStorage() {
	const ls = window.localStorage;
	if (typeof ls?.clear === "function") {
		ls.clear();
		return;
	}
	for (let i = ls.length - 1; i >= 0; i--) {
		const k = ls.key(i);
		if (k) ls.removeItem(k);
	}
}

beforeEach(() => {
	clearLocalStorage();

	Object.defineProperty(window, "matchMedia", {
		writable: true,
		value: vi.fn().mockImplementation((query: string) => ({
			matches: false,
			media: query,
			onchange: null,
			addListener: vi.fn(),
			removeListener: vi.fn(),
			addEventListener: vi.fn(),
			removeEventListener: vi.fn(),
			dispatchEvent: vi.fn(),
		})),
	});
});

vi.mock("../hooks/useAuth", () => ({
	useAuth: () => ({ isAuthenticated: true }),
}));

const mockCreateReindexJob = vi.fn();
const mockGetReindexJob = vi.fn();
const mockListReindexJobs = vi.fn();
const mockStopReindexJob = vi.fn();
const mockResumeReindexJob = vi.fn();
const mockAuditSearch = vi.fn();
vi.mock("../api/admin", () => ({
	adminApi: {
		createReindexJob: (...args: unknown[]) => mockCreateReindexJob(...args),
		getReindexJob: (...args: unknown[]) => mockGetReindexJob(...args),
		listReindexJobs: (...args: unknown[]) => mockListReindexJobs(...args),
		stopReindexJob: (...args: unknown[]) => mockStopReindexJob(...args),
		resumeReindexJob: (...args: unknown[]) => mockResumeReindexJob(...args),
		auditSearch: (...args: unknown[]) => mockAuditSearch(...args),
	},
}));

vi.mock("../api/search", () => ({
	searchApi: {
		fetchSyncStatus: vi.fn(() =>
			Promise.resolve({
				elasticsearch_ok: true,
				outbox_relay_enabled: true,
				outbox_es_subscriber_enabled: true,
				search_index_mode: "outbox_es_subscriber",
			}),
		),
		fetchSyncProgress: vi.fn(() =>
			Promise.resolve({
				postgres_assets_total: 1000,
				elasticsearch_docs_total: 980,
				pg_es_gap: 20,
				pg_es_sync_ratio: 0.98,
				outbox_pending_events: 10,
				outbox_pending_claimable: 3,
				outbox_processing_events: 2,
				outbox_relay_safety_lag_sec: 2,
				oldest_pending_age_sec: 3,
				checked_at: "2026-05-12T00:00:00Z",
			}),
		),
	},
}));

vi.mock("../lib/apiError", () => ({
	extractApiErrorMessage: (_err: unknown, fallback: string) => fallback,
}));

afterEach(() => {
	cleanup();
	mockCreateReindexJob.mockReset();
	mockGetReindexJob.mockReset();
	mockListReindexJobs.mockReset();
	mockStopReindexJob.mockReset();
	mockResumeReindexJob.mockReset();
	mockAuditSearch.mockReset();
	mockListReindexJobs.mockResolvedValue([]);
});

beforeEach(() => {
	mockListReindexJobs.mockResolvedValue([]);
});

describe("SettingsPage", () => {
	it("renders the settings page with session info", () => {
		render(<SettingsPage />);
		expect(screen.getByText("设置")).toBeTruthy();
		expect(screen.getByText("当前会话")).toBeTruthy();
		expect(screen.getByText("Cookie Session (HttpOnly)")).toBeTruthy();
	});

	it("renders the reindex card with single rebuild button", () => {
		render(<SettingsPage />);
		expect(screen.getAllByText("重建 ES 索引").length).toBeGreaterThan(0);
		expect(screen.getByTestId("reindex-open-confirm-btn")).toBeTruthy();
	});

	it("opens confirmation modal showing audit data when rebuild is clicked", async () => {
		mockAuditSearch.mockResolvedValue({
			pg_assets: 268797,
			elasticsearch_docs: 2400,
			missing_in_elasticsearch: 266397,
			orphan_in_elasticsearch: 0,
			consistency: 0.0089,
			target: 0.999,
			duration_ms: 67,
		});

		render(<SettingsPage />);
		fireEvent.click(screen.getByTestId("reindex-open-confirm-btn"));

		await waitFor(() => {
			expect(mockAuditSearch).toHaveBeenCalled();
			expect(screen.getByText(/268,?797/)).toBeTruthy();
			expect(screen.getByText(/266,?397/)).toBeTruthy();
		});
	});

	it("creates a non-dry reindex job when confirmation is submitted", async () => {
		mockAuditSearch.mockResolvedValue({
			pg_assets: 100,
			elasticsearch_docs: 80,
			missing_in_elasticsearch: 20,
			orphan_in_elasticsearch: 0,
			consistency: 0.8,
			target: 0.999,
			duration_ms: 5,
		});
		mockCreateReindexJob.mockResolvedValue({
			id: "rj_run",
			status: "queued",
			dry_run: false,
			page_size: 200,
			next_page: 1,
			stop_requested: false,
			total_assets: 0,
			assets_scanned: 0,
			documents_indexed: 0,
			documents_deleted: 0,
			failed: 0,
			progress_pct: 0,
			created_at: "2026-05-13T00:00:00Z",
			updated_at: "2026-05-13T00:00:00Z",
		});
		mockGetReindexJob.mockResolvedValue({
			id: "rj_run",
			status: "running",
			dry_run: false,
			page_size: 200,
			next_page: 2,
			stop_requested: false,
			total_assets: 100,
			assets_scanned: 25,
			documents_indexed: 25,
			documents_deleted: 0,
			failed: 0,
			progress_pct: 25,
			created_at: "2026-05-13T00:00:00Z",
			updated_at: "2026-05-13T00:00:02Z",
		});

		render(<SettingsPage />);
		fireEvent.click(screen.getByTestId("reindex-open-confirm-btn"));

		await waitFor(() => {
			expect(screen.getByText(/100/)).toBeTruthy();
		});
		fireEvent.click(screen.getByText("确认重建"));

		await waitFor(() => {
			expect(mockCreateReindexJob).toHaveBeenCalledWith(false);
			expect(screen.getByTestId("reindex-stop-btn")).toBeTruthy();
		});
	});

	it("adopts an active non-dry job from the server when idle", async () => {
		mockListReindexJobs.mockResolvedValue([
			{
				id: "rj_active",
				status: "running",
				dry_run: false,
				page_size: 200,
				next_page: 5,
				stop_requested: false,
				total_assets: 1000,
				assets_scanned: 800,
				documents_indexed: 800,
				documents_deleted: 0,
				failed: 0,
				progress_pct: 80,
				created_at: "2026-05-13T00:00:00Z",
				updated_at: "2026-05-13T00:00:30Z",
			},
		]);
		mockGetReindexJob.mockResolvedValue({
			id: "rj_active",
			status: "running",
			dry_run: false,
			page_size: 200,
			next_page: 6,
			stop_requested: false,
			total_assets: 1000,
			assets_scanned: 900,
			documents_indexed: 900,
			documents_deleted: 0,
			failed: 0,
			progress_pct: 90,
			created_at: "2026-05-13T00:00:00Z",
			updated_at: "2026-05-13T00:00:35Z",
		});

		render(<SettingsPage />);

		await waitFor(() => {
			expect(mockGetReindexJob).toHaveBeenCalledWith("rj_active");
			expect(screen.getByTestId("reindex-stop-btn")).toBeTruthy();
		});
	});
});
