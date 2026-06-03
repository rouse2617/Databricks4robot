import {
	ArrowLeftOutlined,
	FileOutlined,
	LinkOutlined,
	SendOutlined,
	TagOutlined,
} from "@ant-design/icons";
import {
	Button,
	Empty,
	message,
	Result,
	Spin,
	Tabs,
	Tag,
	Typography,
} from "antd";
import { lazy, Suspense, useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import { assetsApi } from "../api/assets";
import {
	type AssetMetricItem,
	type EvalResultItem,
	evalApi,
} from "../api/eval";
import type { AlgoEvent, AlgoStatus, Asset, AssetEvent } from "../api/types";
import AssetPreviewHero from "../components/asset-detail/AssetPreviewHero";

const ActionsTimelineTab = lazy(
	() => import("../components/asset-detail/ActionsTimelineTab"),
);
const AlgoTab = lazy(() => import("../components/asset-detail/AlgoTab"));
const AssetEventsTab = lazy(
	() => import("../components/asset-detail/AssetEventsTab"),
);
const DeliveryHistoryTab = lazy(
	() => import("../components/asset-detail/DeliveryHistoryTab"),
);
const EvalMetricsTab = lazy(
	() => import("../components/asset-detail/EvalMetricsTab"),
);
const FilesTab = lazy(() => import("../components/asset-detail/FilesTab"));
const LineageTab = lazy(() => import("../components/asset-detail/LineageTab"));

import OverviewTab from "../components/asset-detail/OverviewTab";

const TagsTab = lazy(() => import("../components/asset-detail/TagsTab"));

const TAB_FALLBACK = (
	<div style={{ padding: 24, textAlign: "center" }}>加载中...</div>
);

import { buildPreviewManifestFromSources } from "../hooks/assets/useAssetPreview";
import { extractApiErrorMessage } from "../lib/apiError";
import {
	getAssetStateColor,
	getLifecycleState,
} from "../lib/assetPresentation";
import {
	type AssetDetailLocationState,
	clearStoredAssetDetailReturn,
	consumeStoredReturnUrl,
	isSafeInternalReturnUrl,
} from "../lib/assets/assetWorkbenchNavigation";

const { Title, Text } = Typography;

/** Parse algo_results map into structured algo info list. */
function parseAlgoResults(algoResults: Record<string, string> | undefined) {
	if (!algoResults) return [];
	const algos = new Map<string, Record<string, string>>();
	for (const [k, v] of Object.entries(algoResults)) {
		const colonIdx = k.indexOf(":");
		if (colonIdx === -1) continue;
		const algoKey = k.substring(0, colonIdx);
		const field = k.substring(colonIdx + 1);
		if (!algos.has(algoKey)) algos.set(algoKey, {});
		const next = algos.get(algoKey);
		if (next) {
			next[field] = v;
		}
	}
	return Array.from(algos.entries()).map(([key, fields]) => ({
		key,
		name: key.split("@")[0],
		version: key.split("@")[1] ?? "",
		status: (fields.status ?? "pending") as AlgoStatus,
		started_at: fields.started_at ?? fields.at,
		finished_at: fields.finished_at,
		method: fields.method,
		run_id: fields.run_id,
		output_uri: fields.output_uri,
		reason: fields.reason,
	}));
}

export default function AssetDetailPage() {
	const { id } = useParams<{ id: string }>();
	const navigate = useNavigate();
	const location = useLocation();
	const returnTo = (location.state as AssetDetailLocationState | null)
		?.assetsReturnTo;
	const [asset, setAsset] = useState<Asset | null>(null);
	const [assetError, setAssetError] = useState<string | null>(null);
	const [loading, setLoading] = useState(true);
	const [algoEvents, setAlgoEvents] = useState<AlgoEvent[]>([]);
	const [algoEventsCursor, setAlgoEventsCursor] = useState<number | null>(null);
	const [algoEventsLoading, setAlgoEventsLoading] = useState(false);
	const [allEvents, setAllEvents] = useState<AssetEvent[]>([]);
	const [allEventsCursor, setAllEventsCursor] = useState<number | null>(null);
	const [allEventsLoading, setAllEventsLoading] = useState(false);
	const [evalResults, setEvalResults] = useState<EvalResultItem[]>([]);
	const [assetMetrics, setAssetMetrics] = useState<AssetMetricItem[]>([]);
	const [evalLoading, setEvalLoading] = useState(false);
	const [msg, msgCtx] = message.useMessage();
	const [previewManifest, setPreviewManifest] = useState<ReturnType<
		typeof buildPreviewManifestFromSources
	> | null>(null);
	const [didInitialLoad, setDidInitialLoad] = useState(false);

	const jumpToPreviewTime = useCallback(
		(eventTimeIso: string) => {
			if (!id) return;
			const millis = Date.parse(eventTimeIso);
			if (!Number.isFinite(millis) || millis < 0) {
				msg.error("事件时间无效，无法定位预览");
				return;
			}
			const sec = millis / 1000;
			const base =
				returnTo && isSafeInternalReturnUrl(returnTo)
					? returnTo
					: `/assets?preview=${encodeURIComponent(id)}`;
			const parsed = new URL(base, "http://local");
			parsed.searchParams.set("preview", id);
			parsed.searchParams.set("preview_time", String(sec));
			parsed.searchParams.set("time", String(sec));
			navigate(`${parsed.pathname}?${parsed.searchParams.toString()}`);
		},
		[id, msg, navigate, returnTo],
	);

	const loadAsset = useCallback(
		(options?: { background?: boolean }) => {
			if (!id) return;
			const background = options?.background ?? false;
			if (!background) {
				setLoading(true);
			}
			setAssetError(null);
			assetsApi
				.get(id)
				.then(async (nextAsset) => {
					setAsset(nextAsset);
					const foxgloveSource = await assetsApi
						.getFoxgloveSource(id)
						.catch(() => null);
					setPreviewManifest(
						buildPreviewManifestFromSources(nextAsset, null, foxgloveSource),
					);
				})
				.catch((err) => {
					const nextError = extractApiErrorMessage(err, "加载资产失败");
					setAssetError(nextError);
					msg.error(nextError);
				})
				.finally(() => {
					if (!background) {
						setLoading(false);
						setDidInitialLoad(true);
					}
				});
		},
		[id, msg],
	);

	const loadAlgoEvents = useCallback(
		(cursor?: number) => {
			if (!id) return;
			setAlgoEventsLoading(true);
			assetsApi
				.listAlgoEvents(id, undefined, cursor, 20)
				.then((resp) => {
					setAlgoEvents((prev) =>
						cursor ? [...prev, ...(resp.items ?? [])] : (resp.items ?? []),
					);
					setAlgoEventsCursor(resp.next_cursor ?? null);
				})
				.catch(() => {
					if (!cursor) setAlgoEvents([]);
				})
				.finally(() => setAlgoEventsLoading(false));
		},
		[id],
	);

	const loadAllEvents = useCallback(
		(cursor?: number) => {
			if (!id) return;
			setAllEventsLoading(true);
			assetsApi
				.listEvents(id, cursor ? { cursor, limit: 20 } : { limit: 20 })
				.then((resp) => {
					setAllEvents((prev) =>
						cursor ? [...prev, ...(resp.items ?? [])] : (resp.items ?? []),
					);
					setAllEventsCursor(resp.next_cursor ?? null);
				})
				.catch(() => {
					if (!cursor) setAllEvents([]);
				})
				.finally(() => setAllEventsLoading(false));
		},
		[id],
	);

	const refresh = useCallback(() => {
		loadAsset({ background: false });
		loadAlgoEvents();
		loadAllEvents();
		if (id) {
			setEvalLoading(true);
			Promise.all([evalApi.listEvalResults(id, 20), evalApi.listMetrics(id)])
				.then(([evalResp, metricsResp]) => {
					setEvalResults(evalResp.items ?? []);
					setAssetMetrics(metricsResp.items ?? []);
				})
				.catch(() => {
					setEvalResults([]);
					setAssetMetrics([]);
				})
				.finally(() => setEvalLoading(false));
		}
	}, [id, loadAlgoEvents, loadAllEvents, loadAsset]);

	const refreshAfterTagUpdate = useCallback(() => {
		loadAsset({ background: true });
		loadAllEvents();
	}, [loadAllEvents, loadAsset]);

	useEffect(() => {
		setDidInitialLoad(false);
		setAsset(null);
		setAssetError(null);
		setAlgoEvents([]);
		setAlgoEventsCursor(null);
		setAllEvents([]);
		setAllEventsCursor(null);
		setEvalResults([]);
		setAssetMetrics([]);
		setPreviewManifest(null);
		refresh();
	}, [refresh]);

	if (loading && !didInitialLoad) {
		return (
			<div
				className="flex items-center justify-center"
				style={{ height: "60vh" }}
			>
				<Spin size="large" />
			</div>
		);
	}
	if (!asset) {
		if (assetError) {
			return (
				<Result
					status="error"
					title="资产加载失败"
					subTitle={assetError}
					extra={
						<>
							<Button type="primary" onClick={refresh}>
								重试
							</Button>
							<Button onClick={() => navigate("/assets")}>返回资产列表</Button>
						</>
					}
				/>
			);
		}
		return <Empty description="资产未找到" />;
	}

	const algoList = parseAlgoResults(asset.algo_results);

	const tabItems = [
		{
			key: "overview",
			label: "概览",
			children: <OverviewTab asset={asset} />,
		},
		{
			key: "algo",
			label: algoEventsLoading
				? "算法处理 (...)"
				: `算法处理 (${algoList.length})`,
			children: (
				<Suspense fallback={TAB_FALLBACK}>
					<AlgoTab
						assetId={asset.asset_id}
						algoList={algoList}
						events={algoEvents}
						eventsLoading={algoEventsLoading}
						hasMoreEvents={algoEventsCursor !== null}
						onLoadMoreEvents={() => {
							if (algoEventsCursor !== null) loadAlgoEvents(algoEventsCursor);
						}}
						onRefresh={refresh}
						onJumpToPreviewTime={jumpToPreviewTime}
					/>
				</Suspense>
			),
		},
		{
			key: "events",
			label: allEventsLoading
				? "全部事件 (...)"
				: `全部事件 (${allEvents.length})`,
			children: (
				<Suspense fallback={TAB_FALLBACK}>
					<AssetEventsTab
						assetId={asset.asset_id}
						events={allEvents}
						loading={allEventsLoading}
						hasMore={allEventsCursor !== null}
						onLoadMore={() => {
							if (allEventsCursor !== null) loadAllEvents(allEventsCursor);
						}}
						onJumpToPreviewTime={jumpToPreviewTime}
					/>
				</Suspense>
			),
		},
		{
			key: "eval-metrics",
			label: evalLoading
				? "评测与指标 (...)"
				: `评测与指标 (${assetMetrics.length})`,
			children: (
				<Suspense fallback={TAB_FALLBACK}>
					<EvalMetricsTab
						loading={evalLoading}
						evalResults={evalResults}
						metrics={assetMetrics}
						onRefresh={refresh}
					/>
				</Suspense>
			),
		},
		{
			key: "actions",
			label: "Action 时间轴",
			children: (
				<Suspense fallback={TAB_FALLBACK}>
					<ActionsTimelineTab
						assetId={asset.asset_id}
						assetType={asset.asset_type}
						segStartNs={asset.start_timestamp_ns}
						segEndNs={asset.end_timestamp_ns}
					/>
				</Suspense>
			),
		},
		{
			key: "tags",
			label: (
				<span>
					<TagOutlined /> 标签
				</span>
			),
			children: (
				<Suspense fallback={TAB_FALLBACK}>
					<TagsTab
						assetId={asset.asset_id}
						tags={asset.tags ?? {}}
						onUpdate={refreshAfterTagUpdate}
					/>
				</Suspense>
			),
		},
		{
			key: "deliveries",
			label: (
				<span>
					<SendOutlined /> 交付历史
				</span>
			),
			children: (
				<Suspense fallback={TAB_FALLBACK}>
					<DeliveryHistoryTab assetId={asset.asset_id} />
				</Suspense>
			),
		},
		{
			key: "lineage",
			label: (
				<span>
					<LinkOutlined /> 血缘
				</span>
			),
			children: (
				<Suspense fallback={TAB_FALLBACK}>
					<LineageTab assetId={asset.asset_id} />
				</Suspense>
			),
		},
		{
			key: "files",
			label: (
				<span>
					<FileOutlined /> 文件
				</span>
			),
			children: (
				<Suspense fallback={TAB_FALLBACK}>
					<FilesTab files={asset.files ?? {}} />
				</Suspense>
			),
		},
	];

	return (
		<div>
			{msgCtx}

			{/* Header */}
			<div className="flex items-center gap-3 mb-4">
				<Button
					icon={<ArrowLeftOutlined />}
					onClick={() => {
						if (returnTo && isSafeInternalReturnUrl(returnTo)) {
							clearStoredAssetDetailReturn();
							navigate(returnTo);
							return;
						}
						const stored = consumeStoredReturnUrl();
						if (stored) {
							navigate(stored);
							return;
						}
						if (typeof window !== "undefined" && window.history.length > 1) {
							navigate(-1);
							return;
						}
						navigate("/assets");
					}}
					size="small"
					aria-label="返回"
				/>
				<Title level={4} style={{ margin: 0 }}>
					资产详情
				</Title>
				<Tag color={getAssetStateColor(asset)}>
					{getLifecycleState(asset) || "—"}
				</Tag>
				<Text type="secondary" className="text-xs font-mono">
					{asset.asset_id}
				</Text>
				<Button
					size="small"
					onClick={() => {
						if (asset?.asset_id) {
							navigate(
								`/pipeline?tab=pipelines&asset_ids=${encodeURIComponent(asset.asset_id)}`,
							);
						}
					}}
				>
					用此资产运行 Pipeline
				</Button>
			</div>

			{/* Preview Hero */}
			<AssetPreviewHero asset={asset} previewManifest={previewManifest} />

			{/* Tabs */}
			<Tabs
				defaultActiveKey="overview"
				items={tabItems}
				size="small"
				style={{ marginTop: -8 }}
			/>
		</div>
	);
}
