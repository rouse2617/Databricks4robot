import { Segmented } from "antd";
import { useEffect, useMemo, useState } from "react";
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

export function ExecutionRecordsPanel({ active = true }: ExecutionRecordsPanelProps) {
	const [searchParams, setSearchParams] = useSearchParams();
	const executionView = useMemo(
		() => parseExecutionView(searchParams.get("executionView")),
		[searchParams],
	);
	const [view, setView] = useState<ExecutionView>(executionView);

	useEffect(() => {
		setView(executionView);
	}, [executionView]);

	const setExecutionView = (next: ExecutionView) => {
		setView(next);
		const params = new URLSearchParams(searchParams);
		if (next === "batch") params.set("executionView", "batch");
		else params.delete("executionView");
		setSearchParams(params, { replace: true });
	};

	return (
		<div className="pipeline-execution-records">
			<Segmented
				value={view}
				onChange={(value) => setExecutionView(value as ExecutionView)}
				options={[
					{ label: "单次执行", value: "single" },
					{ label: "批量任务", value: "batch" },
				]}
				style={{ marginBottom: 16 }}
			/>
			{view === "single" ? (
				<WorkflowExecutionList active={active} embedded />
			) : (
				<BatchJobList active={active} />
			)}
		</div>
	);
}
