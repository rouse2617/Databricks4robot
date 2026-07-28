// CYB-4294b + CYB-4306: assets workbench container. `/assets` used to be
// just the discovery/list view; CYB-4294 added a batch duration-lookup UI
// and CYB-4306 added a batch cost-lookup UI. Each view is lazy-loaded so
// switching tabs pays the split-chunk cost only for the view the user
// actually opens.
//
//   /assets                  → view=list (default)
//   /assets?view=durations   → view=durations
//   /assets?view=costs       → view=costs

import {
	ClockCircleOutlined,
	DollarOutlined,
	TableOutlined,
} from "@ant-design/icons";
import { Spin, Tabs } from "antd";
import type { ReactElement } from "react";
import { lazy, Suspense } from "react";
import { useSearchParams } from "react-router-dom";

function TabLoader() {
	return (
		<div
			className="flex items-center justify-center"
			style={{ height: "40vh" }}
		>
			<Spin size="large" />
		</div>
	);
}

const AssetsPage = lazy(() => import("./AssetsPage"));
const AssetDurationLookup = lazy(() => import("./AssetDurationLookup"));
const AssetCostsLookup = lazy(() => import("./AssetCostsLookup"));

const VIEW_KEYS = ["list", "durations", "costs"] as const;
type ViewKey = (typeof VIEW_KEYS)[number];

function isViewKey(v: string | null): v is ViewKey {
	return v !== null && (VIEW_KEYS as readonly string[]).includes(v);
}

export default function AssetsWorkbenchPage() {
	const [params, setParams] = useSearchParams();
	const raw = params.get("view");
	const view: ViewKey = isViewKey(raw) ? raw : "list";

	function handleChange(next: string): void {
		setParams(
			(prev) => {
				const out = new URLSearchParams(prev);
				if (next === "list") {
					out.delete("view");
				} else {
					out.set("view", next);
				}
				return out;
			},
			// replace so tab clicks don't flood browser history.
			{ replace: true },
		);
	}

	let body: ReactElement;
	if (view === "durations") body = <AssetDurationLookup />;
	else if (view === "costs") body = <AssetCostsLookup />;
	else body = <AssetsPage />;

	return (
		<div>
			<div style={{ padding: "12px 24px 0" }}>
				<Tabs
					activeKey={view}
					onChange={handleChange}
					items={[
						{
							key: "list",
							label: (
								<span>
									<TableOutlined /> 资产列表
								</span>
							),
						},
						{
							key: "durations",
							label: (
								<span>
									<ClockCircleOutlined /> 时长批量查询
								</span>
							),
						},
						{
							key: "costs",
							label: (
								<span>
									<DollarOutlined /> 成本查询
								</span>
							),
						},
					]}
				/>
			</div>
			<Suspense fallback={<TabLoader />}>{body}</Suspense>
		</div>
	);
}
