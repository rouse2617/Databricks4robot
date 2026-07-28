// CYB-4294b: assets workbench container. `/assets` used to be just the
// discovery/list view; CYB-4294 added a batch duration-lookup UI. Initial
// landing put duration-lookup as its own sidebar item ("时长批量查询"), but
// the two views are two lenses on the same asset set, so the sidebar entry
// was dropped and this wrapper renders both under a single tab strip:
//
//   /assets              → view=list (default)
//   /assets?view=durations → view=durations
//
// The wrapper stays deliberately thin. `AssetsPage` still owns the big
// discovery workbench; `AssetDurationLookup` still owns the paste/query UI —
// both are lazy-loaded so switching tabs pays the split-chunk cost only for
// the view the user actually opens.

import { ClockCircleOutlined, TableOutlined } from "@ant-design/icons";
import { Spin, Tabs } from "antd";
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

const VIEW_KEYS = ["list", "durations"] as const;
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
					]}
				/>
			</div>
			<Suspense fallback={<TabLoader />}>
				{view === "durations" ? <AssetDurationLookup /> : <AssetsPage />}
			</Suspense>
		</div>
	);
}
