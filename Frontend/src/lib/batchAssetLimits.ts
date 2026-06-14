/** Maximum assets allowed in one batch backfill job (must match backend). */
export const MAX_BATCH_ASSET_COUNT = 10_000;

export function batchAssetLimitError(count: number): string {
	return `批量任务最多支持 ${MAX_BATCH_ASSET_COUNT.toLocaleString()} 个资产，当前为 ${count.toLocaleString()} 个`;
}

export function exceedsBatchAssetLimit(count: number): boolean {
	return count > MAX_BATCH_ASSET_COUNT;
}
