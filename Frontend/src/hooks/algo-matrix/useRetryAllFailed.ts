// ─── useRetryAllFailed — Batch retry all failed algo tasks on current page ───
// Validates: Requirements REQ-3.3.9

import { useCallback, useState } from "react";
import type { AlgoRegistryItem } from "../../api/algoRegistry";
import { assetsApi } from "../../api/assets";
import type { Asset } from "../../api/types";
import { getAlgoStatusFromResults, isFailedStatus } from "../../lib/algoStatus";

export interface FailedPair {
	assetId: string;
	algoKey: string;
}

export interface RetryResult {
	success: number;
	failed: number;
}

// Collect all (asset_id, algo_key) pairs whose status normalizes to "failed".
// Uses the same parser as the matrix grid so the two views agree on what is
// considered failed (covers raw "error", "failed", JSON {status:"failed"} etc).
export function collectFailedPairs(
	assets: Asset[],
	algorithms: AlgoRegistryItem[],
): FailedPair[] {
	const pairs: FailedPair[] = [];
	for (const asset of assets) {
		for (const algo of algorithms) {
			if (
				isFailedStatus(getAlgoStatusFromResults(asset.algo_results, algo.key))
			) {
				pairs.push({ assetId: asset.asset_id, algoKey: algo.key });
			}
		}
	}
	return pairs;
}

/** Run async tasks with concurrency limit */
async function runWithConcurrency<T>(
	tasks: (() => Promise<T>)[],
	limit: number,
	onProgress: () => void,
): Promise<T[]> {
	const results: T[] = new Array(tasks.length);
	let idx = 0;

	async function worker() {
		while (idx < tasks.length) {
			const i = idx++;
			results[i] = await tasks[i]();
			onProgress();
		}
	}

	await Promise.all(
		Array.from({ length: Math.min(limit, tasks.length) }, () => worker()),
	);
	return results;
}

export function useRetryAllFailed() {
	const [executing, setExecuting] = useState(false);
	const [progress, setProgress] = useState(0);
	const [total, setTotal] = useState(0);

	const execute = useCallback(
		async (pairs: FailedPair[]): Promise<RetryResult> => {
			setExecuting(true);
			setProgress(0);
			setTotal(pairs.length);

			let success = 0;
			let failed = 0;
			let completed = 0;

			const tasks = pairs.map(({ assetId, algoKey }) => async () => {
				try {
					await assetsApi.resetAlgo(assetId, algoKey);
					success++;
				} catch {
					failed++;
				}
			});

			await runWithConcurrency(tasks, 5, () => {
				completed++;
				setProgress(Math.round((completed / pairs.length) * 100));
			});

			setExecuting(false);
			return { success, failed };
		},
		[],
	);

	return { executing, progress, total, execute };
}
