// CYB-4294b + CYB-4306 + CYB-4333: assets workbench container. Each batch
// lookup used to own its own outer tab; CYB-4333 collapsed them behind a
// single "批量查询" tab that hosts a preset picker (see BatchLookupPage).
//
//   /assets                                → view=list (default)
//   /assets?view=lookup                    → BatchLookupPage, default preset
//   /assets?view=lookup&preset=<key>       → BatchLookupPage, specific preset
//   /assets?view=durations                 → redirects to lookup+durations
//   /assets?view=costs                     → redirects to lookup+costs

import { SearchOutlined, TableOutlined } from "@ant-design/icons";
import { Spin, Tabs } from "antd";
import type { ReactElement } from "react";
import { lazy, Suspense } from "react";
import { Navigate, useSearchParams } from "react-router-dom";

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
const BatchLookupPage = lazy(() => import("./BatchLookupPage"));

const VIEW_KEYS = ["list", "lookup"] as const;
type ViewKey = (typeof VIEW_KEYS)[number];

function isViewKey(v: string | null): v is ViewKey {
	return v !== null && (VIEW_KEYS as readonly string[]).includes(v);
}

// Legacy `?view=durations` / `?view=costs` URLs shipped in CYB-4304 /
// CYB-4306 respectively; a redirect (Navigate replace) keeps any stale
// bookmark or Slack link from silently landing on the 资产列表 tab.
const LEGACY_PRESET_VIEWS = ["durations", "costs"] as const;
type LegacyPresetView = (typeof LEGACY_PRESET_VIEWS)[number];

function isLegacyPresetView(v: string | null): v is LegacyPresetView {
	return v !== null && (LEGACY_PRESET_VIEWS as readonly string[]).includes(v);
}

export default function AssetsWorkbenchPage() {
	const [params, setParams] = useSearchParams();
	const raw = params.get("view");

	// Compat redirect BEFORE the tabs render so a stale bookmark doesn't
	// briefly flash the wrong tab state. Preserves any other query params.
	if (isLegacyPresetView(raw)) {
		const target = new URLSearchParams(params);
		target.set("view", "lookup");
		target.set("preset", raw);
		return <Navigate to={`/assets?${target.toString()}`} replace />;
	}

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

	const body: ReactElement =
		view === "lookup" ? <BatchLookupPage /> : <AssetsPage />;

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
							key: "lookup",
							label: (
								<span>
									<SearchOutlined /> 批量查询
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
