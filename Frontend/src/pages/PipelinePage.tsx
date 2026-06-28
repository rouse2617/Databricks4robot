import { App, Spin, Tabs } from "antd";
import { lazy, Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import ErrorBoundary from "../components/ErrorBoundary";

import "../styles/pipeline.css";

type PipelineTab = "design" | "pipelines" | "executions" | "components";

interface PipelinePageProps {
	defaultTab?: PipelineTab;
}

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

const CANONICAL_TABS: ReadonlySet<PipelineTab> = new Set([
	"design",
	"pipelines",
	"executions",
	"components",
]);

// 常见别名 → 规范 tab；命中别名时静默纠正 URL，不打扰用户。
const TAB_ALIASES: Record<string, PipelineTab> = {
	templates: "pipelines",
	template: "pipelines",
	manage: "pipelines",
	saved: "pipelines",
	runs: "executions",
	run: "executions",
	canvas: "design",
	designer: "design",
	component: "components",
	steps: "components",
};

const TAB_DISPLAY_NAMES: Record<PipelineTab, string> = {
	design: "设计",
	pipelines: "流水线",
	executions: "执行记录",
	components: "组件",
};

function resolvePipelineTab(
	raw: string | null,
	defaultTab: PipelineTab = "design",
): PipelineTab {
	if (raw && CANONICAL_TABS.has(raw as PipelineTab)) {
		return raw as PipelineTab;
	}
	if (raw && TAB_ALIASES[raw]) {
		return TAB_ALIASES[raw];
	}
	return defaultTab;
}

export default function PipelinePage({
	defaultTab = "design",
}: PipelinePageProps) {
	const [searchParams, setSearchParams] = useSearchParams();
	const { modal, message } = App.useApp();
	const [designDirty, setDesignDirty] = useState(false);
	const activeTab = useMemo(
		() => resolvePipelineTab(searchParams.get("tab"), defaultTab),
		[defaultTab, searchParams],
	);

	// 处理 URL 里的 tab 参数：别名静默纠正、未识别参数给出提示，
	// 避免「打开 ?tab=xxx 却静默落到设计页」的困惑。
	useEffect(() => {
		const raw = searchParams.get("tab");
		if (!raw || CANONICAL_TABS.has(raw as PipelineTab)) return;
		const resolved = resolvePipelineTab(raw, defaultTab);
		const isKnownAlias = Boolean(TAB_ALIASES[raw]);
		if (!isKnownAlias) {
			message.warning(
				`未识别的标签参数 “${raw}”，已为你切换到「${TAB_DISPLAY_NAMES[resolved]}」`,
			);
		}
		const next = new URLSearchParams(searchParams);
		if (resolved === defaultTab) {
			next.delete("tab");
		} else {
			next.set("tab", resolved);
		}
		setSearchParams(next, { replace: true });
	}, [searchParams, defaultTab, message, setSearchParams]);
	const onTabChange = useCallback(
		async (nextTab: string) => {
			if (nextTab !== activeTab && designDirty) {
				const confirmed = await confirmLeaveWithUnsavedChanges(modal);
				if (!confirmed) return;
				setDesignDirty(false);
			}
			const next = new URLSearchParams(searchParams);
			if (nextTab === defaultTab) {
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
		[activeTab, defaultTab, designDirty, modal, searchParams, setSearchParams],
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
