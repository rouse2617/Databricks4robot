import { Alert, Segmented } from "antd";
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

export function ExecutionRecordsPanel({
	active = true,
}: ExecutionRecordsPanelProps) {
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
			<Alert
				type="info"
				showIcon
				closable
				message="执行记录说明"
				description="流水线模板是设计稿；单次执行为一个资产的一次 Workflow 运行；批量任务将多资产打包，可在批次详情查看节点汇总与子任务。"
				style={{ marginBottom: 16 }}
			/>
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
