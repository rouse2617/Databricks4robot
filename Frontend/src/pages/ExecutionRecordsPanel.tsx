import { Segmented, Spin } from "antd";
import { lazy, Suspense, useMemo } from "react";
import { useSearchParams } from "react-router-dom";

const BatchJobList = lazy(async () => {
	const mod = await import("./BatchJobList");
	return { default: mod.BatchJobList };
});

const WorkflowExecutionList = lazy(async () => {
	const mod = await import("./WorkflowExecutionList");
	return { default: mod.WorkflowExecutionList };
});

type ExecutionView = "single" | "batch";

interface ExecutionRecordsPanelProps {
	active?: boolean;
}

function parseExecutionView(value: string | null): ExecutionView {
	return value === "batch" ? "batch" : "single";
}

function ExecutionViewFallback() {
	return (
		<div
			style={{
				minHeight: 220,
				display: "flex",
				alignItems: "center",
				justifyContent: "center",
			}}
		>
			<Spin size="small" />
		</div>
	);
}

export function ExecutionRecordsPanel({
	active = true,
}: ExecutionRecordsPanelProps) {
	const [searchParams, setSearchParams] = useSearchParams();
	const executionView = useMemo(
		() => parseExecutionView(searchParams.get("executionView")),
		[searchParams],
	);

	const setExecutionView = (next: ExecutionView) => {
		const params = new URLSearchParams(searchParams);
		params.set("tab", "executions");
		if (next === "batch") params.set("executionView", "batch");
		else params.delete("executionView");
		setSearchParams(params, { replace: true });
	};

	return (
		<div className="pipeline-execution-records">
			<Segmented
				value={executionView}
				onChange={(value) => setExecutionView(value as ExecutionView)}
				options={[
					{ label: "单次执行", value: "single" },
					{ label: "批量任务", value: "batch" },
				]}
				style={{ marginBottom: 16 }}
			/>
			<Suspense fallback={<ExecutionViewFallback />}>
				{executionView === "single" ? (
					<WorkflowExecutionList active={active} embedded />
				) : (
					<BatchJobList active={active} />
				)}
			</Suspense>
		</div>
	);
}
