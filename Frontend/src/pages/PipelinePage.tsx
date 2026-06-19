import { App, Spin, Tabs } from "antd";
import { lazy, Suspense, useCallback, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import ErrorBoundary from "../components/ErrorBoundary";

import "../styles/pipeline.css";

type PipelineTab = "design" | "pipelines" | "executions" | "components";

const PipelineDesignerTab = lazy(async () => {
	const [{ ReactFlowProvider }, { PipelineDesignerCanvas }] = await Promise.all(
		[
			import("@xyflow/react"),
			import("../features/pipeline-designer/PipelineDesignerCanvas"),
		],
	);

	return {
		default: ({
			onDirtyChange,
		}: {
			onDirtyChange: (dirty: boolean) => void;
		}) => (
			<ReactFlowProvider>
				<PipelineDesignerCanvas onDirtyChange={onDirtyChange} />
			</ReactFlowProvider>
		),
	};
});

const DeployPanel = lazy(async () => {
	const mod = await import("../components/pipeline/DeployPanel");
	return { default: mod.DeployPanel };
});

const ExecutionRecordsPanel = lazy(async () => {
	const mod = await import("./ExecutionRecordsPanel");
	return { default: mod.ExecutionRecordsPanel };
});

const ComponentManager = lazy(async () => {
	const mod = await import("./ComponentManager");
	return { default: mod.ComponentManager };
});

function confirmLeaveWithUnsavedChanges(
	modal: ReturnType<typeof App.useApp>["modal"],
): Promise<boolean> {
	return new Promise((resolve) => {
		modal.confirm({
			title: "离开当前编辑？",
			content: "存在未保存的变更，离开后这些修改会丢失。",
			okText: "离开",
			okType: "danger",
			cancelText: "继续编辑",
			onOk: () => resolve(true),
			onCancel: () => resolve(false),
		});
	});
}

function TabFallback() {
	return (
		<div
			className="pipeline-tab-fallback"
			style={{
				display: "flex",
				alignItems: "center",
				justifyContent: "center",
			}}
		>
			<Spin size="small" />
		</div>
	);
}

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
				destroyOnHidden
				animated={{ inkBar: true, tabPane: false }}
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
									<Suspense fallback={<TabFallback />}>
										<PipelineDesignerTab onDirtyChange={setDesignDirty} />
									</Suspense>
								</ErrorBoundary>
							</div>
						),
					},
					{
						key: "pipelines",
						label: tabLabel("流水线", "管理已保存的流水线"),
						children: (
							<div className="pipeline-tab-content pipeline-tab-content--panel pipeline-tab-content--management">
								<Suspense fallback={<TabFallback />}>
									<DeployPanel variant="full" />
								</Suspense>
							</div>
						),
					},
					{
						key: "executions",
						label: tabLabel("执行记录", "查看和管理流水线运行"),
						children: (
							<div className="pipeline-tab-content pipeline-tab-content--panel pipeline-tab-content--executions">
								<Suspense fallback={<TabFallback />}>
									<ExecutionRecordsPanel active={activeTab === "executions"} />
								</Suspense>
							</div>
						),
					},
					{
						key: "components",
						label: tabLabel("组件", "管理可复用的步骤定义"),
						children: (
							<div className="pipeline-tab-content pipeline-tab-content--panel">
								<Suspense fallback={<TabFallback />}>
									<ComponentManager />
								</Suspense>
							</div>
						),
					},
				]}
			/>
		</div>
	);
}
