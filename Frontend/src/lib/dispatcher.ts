// CYB-3679 — dispatcher tuning display + validation logic (kept pure for tests).

/** Validation ranges — mirror backend validateDispatcherConfig. */
export const DISPATCHER_LIMITS = {
	maxConcurrency: { min: 1, max: 256 },
	submitBatch: { min: 1, max: 200 },
	ratePerSec: { min: 0.1, max: 100 },
} as const;

export interface DispatcherFormValues {
	max_concurrency: number;
	submit_batch: number;
	rate_per_sec: number;
}

/** Returns human-readable errors ([] = valid). Mirrors backend ranges so a
 *  fat-fingered value is rejected before it ever leaves the browser. */
export function validateDispatcherForm(v: DispatcherFormValues): string[] {
	const errs: string[] = [];
	const L = DISPATCHER_LIMITS;
	if (
		!Number.isFinite(v.max_concurrency) ||
		v.max_concurrency < L.maxConcurrency.min ||
		v.max_concurrency > L.maxConcurrency.max
	) {
		errs.push(
			`并发上限须在 ${L.maxConcurrency.min}–${L.maxConcurrency.max} 之间`,
		);
	}
	if (
		!Number.isFinite(v.submit_batch) ||
		v.submit_batch < L.submitBatch.min ||
		v.submit_batch > L.submitBatch.max
	) {
		errs.push(`单轮批量须在 ${L.submitBatch.min}–${L.submitBatch.max} 之间`);
	}
	if (
		!Number.isFinite(v.rate_per_sec) ||
		v.rate_per_sec < L.ratePerSec.min ||
		v.rate_per_sec > L.ratePerSec.max
	) {
		errs.push(`速率须在 ${L.ratePerSec.min}–${L.ratePerSec.max} 个/秒 之间`);
	}
	return errs;
}

/** Suppression chip: label + antd tag color ("" = healthy, no chip). */
export function suppressionChip(
	reason: string | undefined,
): { label: string; color: string } | null {
	switch (reason) {
		case "paused":
			return { label: "已暂停", color: "red" };
		case "aimd_backoff":
			return { label: "背压降速中", color: "orange" };
		default:
			return null;
	}
}

/** DLQ failure reason in plain language (CYB-3679: 人话显示). */
export function humanizeDlqReason(message: string | undefined): string {
	const msg = (message ?? "").toLowerCase();
	if (!msg) return "未知错误";
	if (msg.includes("exceeding the maximum") || msg.includes("steps")) {
		return "流水线步骤数超上限,请拆分批次";
	}
	if (msg.includes("不支持") || msg.includes("not supported")) {
		return "资源规格超出集群上限,请调低模板资源";
	}
	if (msg.includes("template") && msg.includes("not found")) {
		return "模板不存在或已删除";
	}
	if (msg.includes("asset") && msg.includes("not found")) {
		return "资产不存在或已删除";
	}
	if (msg.includes("transpile") || msg.includes("invalid pipeline")) {
		return "流水线定义不合法(编译失败)";
	}
	if (msg.includes("max submit attempts")) {
		return "重试次数用尽(集群持续异常)";
	}
	return message ?? "未知错误";
}
