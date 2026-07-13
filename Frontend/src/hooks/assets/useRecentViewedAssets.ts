// CYB-3382 #5 — 最近查看过的资产(LRU,最多 5 项,localStorage 持久化)
// 消费者:AssetQuickPreviewPane 未选中状态的 empty state 会列出该数组,
// 让用户在没点行时也能快速回访之前看过的资产。

import { useCallback, useEffect, useState } from "react";

const STORAGE_KEY = "assets:recent-viewed";
const MAX_ITEMS = 5;

function readStorage(): string[] {
	if (typeof window === "undefined") return [];
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (!raw) return [];
		const parsed: unknown = JSON.parse(raw);
		if (!Array.isArray(parsed)) return [];
		return parsed.filter((v): v is string => typeof v === "string");
	} catch {
		return [];
	}
}

function writeStorage(ids: string[]): void {
	if (typeof window === "undefined") return;
	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(ids));
	} catch {
		/* localStorage disabled (private mode, quota) — best effort */
	}
}

export interface UseRecentViewedAssetsResult {
	recent: string[];
	pushRecent: (id: string) => void;
	clearRecent: () => void;
}

export function useRecentViewedAssets(): UseRecentViewedAssetsResult {
	const [recent, setRecent] = useState<string[]>(() => readStorage());

	// Sync across tabs — if another tab updates the list, reflect here.
	useEffect(() => {
		if (typeof window === "undefined") return;
		const handler = (e: StorageEvent) => {
			if (e.key === STORAGE_KEY) {
				setRecent(readStorage());
			}
		};
		window.addEventListener("storage", handler);
		return () => window.removeEventListener("storage", handler);
	}, []);

	const pushRecent = useCallback((id: string) => {
		if (!id) return;
		setRecent((prev) => {
			// LRU: 去重 + 置顶 + cap to MAX_ITEMS
			const next = [id, ...prev.filter((existing) => existing !== id)].slice(
				0,
				MAX_ITEMS,
			);
			writeStorage(next);
			return next;
		});
	}, []);

	const clearRecent = useCallback(() => {
		writeStorage([]);
		setRecent([]);
	}, []);

	return { recent, pushRecent, clearRecent };
}

// Exported for testing.
export const RECENT_VIEWED_STORAGE_KEY = STORAGE_KEY;
export const RECENT_VIEWED_MAX_ITEMS = MAX_ITEMS;
