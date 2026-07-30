import { Progress, Tooltip, Typography } from "antd";
import type { BatchJob } from "../../api/batchJobApi";

const { Text } = Typography;

// BatchProgressCell renders a segmented success/failed progress bar so the
// two counts read at a glance instead of a single color that had to be
// re-interpreted per row (CYB batch-list UI refresh). Segments are proportional
// to totalCount so an all-red bar only appears when every child failed —
// mixed batches show a green:red split, and pending capacity is the grey
// trailing region.
export function BatchProgressCell({ job }: { job: BatchJob }) {
	const total = job.totalCount || 0;
	const done = job.completedCount || 0;
	const failed = job.failedCount || 0;
	const pct = (n: number) => (total > 0 ? (n / total) * 100 : 0);
	const donePct = pct(done);
	const failedPct = pct(failed);
	const tip = `成功 ${done} · 失败 ${failed} · 共 ${total}`;

	return (
		<div style={{ minWidth: 0 }}>
			<Tooltip title={tip}>
				<Progress
					size="small"
					// success is stacked before the primary bar; we use it for the
					// green completed count and the primary strokeColor for failed —
					// so the whole "percent" is completed+failed (i.e. finished
					// portion), split by color, with the trailing grey being pending.
					percent={donePct + failedPct}
					success={{ percent: donePct, strokeColor: "#52c41a" }}
					// CYB-4470: failed segment uses desaturated dark red
					// (#cf1322) instead of saturated red (#ff4d4f) to reduce
					// "red alert fatigue" when several rows are all-failed.
					strokeColor="#cf1322"
					showInfo={false}
				/>
			</Tooltip>
			<Text type="secondary" style={{ fontSize: 12 }}>
				{done} 成功 · {failed} 失败 · 共 {total}
			</Text>
		</div>
	);
}
