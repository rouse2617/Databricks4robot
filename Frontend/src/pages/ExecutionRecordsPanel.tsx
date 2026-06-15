import { Segmented } from "antd";
import { useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import { BatchJobList } from "./BatchJobList";
import { WorkflowExecutionList } from "./WorkflowExecutionList";

type ExecutionView = "single" | "batch";

interface ExecutionRecordsPanelProps {
	active?: boolean;
}

function parseExecutionView(value: string | null): ExecutionView {
	return value === "batch" ? "batch" : "single";
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
			{executionView === "single" ? (
				<WorkflowExecutionList active={active} embedded />
			) : (
				<BatchJobList active={active} />
			)}
		</div>
	);
}
