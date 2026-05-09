import {
	ArrowLeftOutlined,
	FileOutlined,
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
import { useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import { assetsApi } from "../api/assets";
import {
	type AssetMetricItem,
	type EvalResultItem,
	evalApi,
} from "../api/eval";
import type { AlgoEvent, AlgoStatus, Asset, AssetEvent } from "../api/types";
import ActionsTimelineTab from "../components/asset-detail/ActionsTimelineTab";
import AlgoTab from "../components/asset-detail/AlgoTab";
import AssetEventsTab from "../components/asset-detail/AssetEventsTab";
import AssetPreviewHero from "../components/asset-detail/AssetPreviewHero";
import DeliveryHistoryTab from "../components/asset-detail/DeliveryHistoryTab";
import EvalMetricsTab from "../components/asset-detail/EvalMetricsTab";
import FilesTab from "../components/asset-detail/FilesTab";
import OverviewTab from "../components/asset-detail/OverviewTab";
import TagsTab from "../components/asset-detail/TagsTab";
import { buildPlaceholderPreviewManifest } from "../hooks/assets/useAssetPreview";
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

	const loadAsset = useCallback(() => {
		if (!id) return;
		setLoading(true);
		setAssetError(null);
		assetsApi
			.get(id)
			.then(setAsset)
			.catch((err) => {
				const nextError = extractApiErrorMessage(err, "加载资产失败");
				setAssetError(nextError);
				msg.error(nextError);
			})
			.finally(() => setLoading(false));
	}, [id, msg]);

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
		loadAsset();
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

	useEffect(() => {
		setAsset(null);
		setAssetError(null);
		setAlgoEvents([]);
		setAlgoEventsCursor(null);
		setAllEvents([]);
		setAllEventsCursor(null);
		setEvalResults([]);
		setAssetMetrics([]);
		refresh();
	}, [refresh]);

	if (loading) {
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
			label: `算法处理 (${algoList.length})`,
			children: (
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
				/>
			),
		},
		{
			key: "events",
			label: `全部事件 (${allEvents.length})`,
			children: (
				<AssetEventsTab
					assetId={asset.asset_id}
					events={allEvents}
					loading={allEventsLoading}
					hasMore={allEventsCursor !== null}
					onLoadMore={() => {
						if (allEventsCursor !== null) loadAllEvents(allEventsCursor);
					}}
				/>
			),
		},
		{
			key: "eval-metrics",
			label: `评测与指标 (${assetMetrics.length})`,
			children: (
				<EvalMetricsTab
					loading={evalLoading}
					evalResults={evalResults}
					metrics={assetMetrics}
					onRefresh={refresh}
				/>
			),
		},
		{
			key: "actions",
			label: "Action 时间轴",
			children: (
				<ActionsTimelineTab
					assetId={asset.asset_id}
					assetType={asset.asset_type}
					segStartNs={asset.start_timestamp_ns}
					segEndNs={asset.end_timestamp_ns}
				/>
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
				<TagsTab
					assetId={asset.asset_id}
					tags={asset.tags ?? {}}
					onUpdate={refresh}
				/>
			),
		},
		{
			key: "deliveries",
			label: (
				<span>
					<SendOutlined /> 交付历史
				</span>
			),
			children: <DeliveryHistoryTab assetId={asset.asset_id} />,
		},
		{
			key: "files",
			label: (
				<span>
					<FileOutlined /> 文件
				</span>
			),
			children: <FilesTab files={asset.files ?? {}} />,
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
			</div>

			{/* Preview Hero */}
			<AssetPreviewHero
				asset={asset}
				previewManifest={buildPlaceholderPreviewManifest(asset)}
			/>

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
