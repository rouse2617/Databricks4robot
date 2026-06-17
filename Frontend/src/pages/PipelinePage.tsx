import { ReactFlowProvider } from "@xyflow/react";
import { App, Tabs } from "antd";
import { useCallback, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import ErrorBoundary from "../components/ErrorBoundary";
import { DeployPanel } from "../components/pipeline/DeployPanel";
import { ComponentManager } from "./ComponentManager";
import { ExecutionRecordsPanel } from "./ExecutionRecordsPanel";
import {
	confirmLeaveWithUnsavedChanges,
	PipelineDesignerCanvas,
} from "../features/pipeline-designer";

import "../styles/pipeline.css";

type PipelineTab = "design" | "pipelines" | "executions" | "components";

function resolvePipelineTab(raw: string | null): PipelineTab {
	switch (raw) {
		case "templates":
		case "pipelines":
			return "pipelines";
		case "executions":
			return "executions";
		case "components":
			return "components";
		default:
			return "design";
	}
}

export default function PipelinePage() {
	const [searchParams, setSearchParams] = useSearchParams();
	const { modal } = App.useApp();
	const [designDirty, setDesignDirty] = useState(false);
	const activeTab = useMemo(
		() => resolvePipelineTab(searchParams.get("tab")),
		[searchParams],
	);
	const onTabChange = useCallback(
		async (nextTab: string) => {
			if (nextTab !== activeTab && designDirty) {
				const confirmed = await confirmLeaveWithUnsavedChanges(modal);
				if (!confirmed) return;
				setDesignDirty(false);
			}
			const next = new URLSearchParams(searchParams);
			if (nextTab === "design") {
				next.delete("tab");
			} else {
				next.set("tab", nextTab);
				next.delete("templateId");
				next.delete("readonly");
				next.delete("asset_ids");
			}
			if (nextTab !== "executions") {
				next.delete("executionView");
			}
			setSearchParams(next, { replace: true });
		},
		[activeTab, designDirty, modal, searchParams, setSearchParams],
	);

	const tabLabel = useCallback((title: string, subtitle: string) => {
		return (
			<div className="pipeline-tab-label">
				<span className="pipeline-tab-label__title">{title}</span>
				<span className="pipeline-tab-label__subtitle">{subtitle}</span>
			</div>
		);
	}, []);

	return (
		<div className="pipeline-tabs-page">
			<Tabs
				activeKey={activeTab}
				onChange={onTabChange}
				destroyOnHidden={false}
				animated={{ inkBar: true, tabPane: true }}
				items={[
					{
						key: "design",
						label: tabLabel("设计", "编辑流水线画布"),
						children: (
							<div className="pipeline-tab-content pipeline-tab-content--canvas">
								<ErrorBoundary
									title="流水线设计错误"
									subTitle="设计画布加载失败，可重试或刷新页面"
								>
									<ReactFlowProvider>
										<PipelineDesignerCanvas onDirtyChange={setDesignDirty} />
									</ReactFlowProvider>
								</ErrorBoundary>
							</div>
						),
					},
					{
						key: "pipelines",
						label: tabLabel("流水线", "管理已保存的流水线"),
						children: (
							<div className="pipeline-tab-content pipeline-tab-content--panel pipeline-tab-content--management">
								<DeployPanel variant="full" />
							</div>
						),
					},
					{
						key: "executions",
						label: tabLabel("执行记录", "查看和管理流水线运行"),
						children: (
							<div className="pipeline-tab-content pipeline-tab-content--panel pipeline-tab-content--executions">
								<ExecutionRecordsPanel active={activeTab === "executions"} />
							</div>
						),
					},
					{
						key: "components",
						label: tabLabel("组件", "管理可复用的步骤定义"),
						children: (
							<div className="pipeline-tab-content pipeline-tab-content--panel">
								<ComponentManager />
							</div>
						),
					},
				]}
			/>
		</div>
	);
}
