import {
	FileOutlined,
	LinkOutlined,
	RobotOutlined,
	SendOutlined,
} from "@ant-design/icons";
import { Button, Card, Descriptions, Empty, Spin, Tag, Typography } from "antd";
import { useEffect, useMemo, useState } from "react";
import { assetsApi } from "../../api/assets";
import type { Asset, AssetProvenance, VersionHistoryEntry } from "../../api/types";
import LogicalAssetId from "./LogicalAssetId";
import RunIdLink from "./RunIdLink";
import { formatDateTime } from "../../lib/dateTime";

const { Text, Title } = Typography;

export interface VersionProvenanceTabProps {
	provenance: AssetProvenance;
	currentAsset: Asset;
	onViewRevision: (assetId: string) => void;
	onOpenLineageTab?: () => void;
}

interface VersionNode {
	version: number;
	assetId: string;
	isCurrent: boolean;
	at: string;
	reason?: string;
	byRunId?: string;
}

function buildVersionNodes(provenance: AssetProvenance): VersionNode[] {
	const revByVersion = new Map(
		provenance.revisions.map((r) => [r.revision, r]),
	);
	const history = [...(provenance.version_history ?? [])].sort(
		(a, b) => a.version - b.version,
	);

	if (history.length > 0) {
		return [...history]
			.map((h: VersionHistoryEntry) => {
				const rev = revByVersion.get(h.version);
				const assetId = h.asset_id ?? rev?.asset_id ?? "";
				return {
					version: h.version,
					assetId,
					isCurrent: rev?.is_current ?? false,
					at: h.promoted_at || rev?.created_at || "",
					reason: h.reason,
					byRunId: h.by_run_id,
				};
			})
			.filter((n) => n.assetId)
			.reverse();
	}

	return [...provenance.revisions]
		.sort((a, b) => b.revision - a.revision)
		.map((r) => ({
			version: r.revision,
			assetId: r.asset_id,
			isCurrent: r.is_current,
			at: r.created_at,
		}));
}

const DIFF_FIELDS: Array<{
	key: keyof Asset;
	label: string;
}> = [
	{ key: "mcap_file_id", label: "MCAP File" },
	{ key: "segment_locator", label: "Segment Locator" },
	{ key: "start_timestamp_ns", label: "起始时间 (ns)" },
	{ key: "end_timestamp_ns", label: "结束时间 (ns)" },
];

function formatFieldValue(value: unknown): string {
	if (value === undefined || value === null || value === "") return "—";
	return String(value);
}

export default function VersionProvenanceTab({
	provenance,
	currentAsset,
	onViewRevision,
	onOpenLineageTab,
}: VersionProvenanceTabProps) {
	const nodes = useMemo(() => buildVersionNodes(provenance), [provenance]);
	const currentRev = provenance.revisions.find((r) => r.is_current);
	const logicalId =
		provenance.logical_asset_id ?? currentAsset.logical_asset_id;

	const [prevAsset, setPrevAsset] = useState<Asset | null>(null);
	const [diffLoading, setDiffLoading] = useState(false);

	const prevRevisionAssetId = useMemo(() => {
		const rev = currentAsset.revision ?? 1;
		if (rev <= 1) return null;
		return (
			provenance.revisions.find((r) => r.revision === rev - 1)?.asset_id ?? null
		);
	}, [currentAsset.revision, provenance.revisions]);

	useEffect(() => {
		if (!prevRevisionAssetId) {
			setPrevAsset(null);
			return;
		}
		let cancelled = false;
		setDiffLoading(true);
		assetsApi
			.get(prevRevisionAssetId)
			.then((a) => {
				if (!cancelled) setPrevAsset(a);
			})
			.catch(() => {
				if (!cancelled) setPrevAsset(null);
			})
			.finally(() => {
				if (!cancelled) setDiffLoading(false);
			});
		return () => {
			cancelled = true;
		};
	}, [prevRevisionAssetId]);

	const diffRows = useMemo(() => {
		if (!prevAsset) return [];
		return DIFF_FIELDS.map(({ key, label }) => {
			const prevVal = formatFieldValue(prevAsset[key]);
			const currVal = formatFieldValue(currentAsset[key]);
			if (prevVal === currVal) return null;
			return { label, prevVal, currVal };
		}).filter((r): r is { label: string; prevVal: string; currVal: string } =>
			Boolean(r),
		);
	}, [prevAsset, currentAsset]);

	const lineage = provenance.lineage;
	const upstreamMcap =
		(lineage?.upstream as { mcap_file_id?: string } | undefined)?.mcap_file_id;
	const downstream = lineage?.downstream as
		| {
				algo_results?: unknown[];
				deliveries?: unknown[];
		  }
		| undefined;
	const algoCount = downstream?.algo_results?.length ?? 0;
	const deliveryCount = downstream?.deliveries?.length ?? 0;

	if (nodes.length === 0) {
		return <Empty description="暂无版本信息" />;
	}

	return (
		<div className="max-w-3xl py-1">
			<Card size="small" className="mb-4">
				<Descriptions column={1} size="small" colon={false}>
					<Descriptions.Item
						label={<Text type="secondary">逻辑资产 ID</Text>}
					>
						{logicalId ? (
							<LogicalAssetId logicalAssetId={logicalId} showLabel={false} />
						) : (
							"—"
						)}
					</Descriptions.Item>
					<Descriptions.Item
						label={<Text type="secondary">当前有效版本</Text>}
					>
						{currentRev ? (
							<span>
								<Tag color="blue" className="mr-2">
									v{currentRev.revision}
								</Tag>
								<Text code className="text-xs">
									{currentRev.asset_id}
								</Text>
							</span>
						) : (
							"—"
						)}
					</Descriptions.Item>
				</Descriptions>
			</Card>

			<Title level={5} style={{ marginTop: 0, marginBottom: 12 }}>
				版本链
			</Title>
			<ol className="list-none m-0 p-0 mb-6" role="list" aria-label="版本链">
				{nodes.map((node, index) => {
					const isViewing = node.assetId === currentAsset.asset_id;
					const isLast = index === nodes.length - 1;
					return (
						<li key={`${node.version}-${node.assetId}`} className="flex gap-3">
							<div className="flex flex-col items-center w-3 shrink-0 pt-1">
								<span
									className={`rounded-full shrink-0 w-2.5 h-2.5 ${node.isCurrent ? "bg-primary" : "border-2 border-text-secondary"}`}
									aria-hidden
								/>
								{!isLast ? (
									<span
										className="flex-1 w-0.5 min-h-[24px] mt-1 bg-border"
										aria-hidden
									/>
								) : null}
							</div>
							<div className={`pb-6 min-w-0 flex-1 ${isLast ? "pb-0" : ""}`}>
								<div className="flex flex-wrap items-center gap-2">
									<Text strong>v{node.version}</Text>
									{node.isCurrent ? (
										<Tag color="blue" className="m-0">
											当前
										</Tag>
									) : null}
									{isViewing && !node.isCurrent ? (
										<Tag className="m-0">正在查看</Tag>
									) : null}
									<Text type="secondary" className="text-xs font-mono">
										{node.assetId}
									</Text>
									<Text type="secondary" className="text-xs">
										{formatDateTime(node.at)}
									</Text>
									{!isViewing ? (
										<Button
											type="link"
											size="small"
											onClick={() => onViewRevision(node.assetId)}
										>
											查看此版
										</Button>
									) : null}
								</div>
								{node.version > 1 && (node.reason || node.byRunId) ? (
									<div className="mt-1 text-xs text-text-secondary">
										{node.reason ? (
											<div>
												升级原因:{" "}
												<Text className="text-xs">{node.reason}</Text>
											</div>
										) : null}
										{node.byRunId ? (
											<div className="flex flex-wrap items-center gap-1">
												<Text type="secondary">触发 run:</Text>
												<RunIdLink runId={node.byRunId} />
											</div>
										) : null}
									</div>
								) : null}
							</div>
						</li>
					);
				})}
			</ol>

			{(currentAsset.revision ?? 1) > 1 ? (
				<>
					<Title level={5} style={{ marginTop: 0, marginBottom: 12 }}>
						与上一版差异
						{prevAsset ? (
							<Text type="secondary" className="text-xs font-normal ml-2">
								v{(currentAsset.revision ?? 1) - 1} → v{currentAsset.revision}
							</Text>
						) : null}
					</Title>
					<Card size="small" className="mb-6">
						{diffLoading ? (
							<Spin size="small" />
						) : diffRows.length === 0 ? (
							<Text type="secondary" className="text-xs">
								{prevAsset
									? "与上一版在关键字段上无差异"
									: "无法加载上一版资产以对比"}
							</Text>
						) : (
							<ul className="m-0 pl-4 text-sm space-y-2">
								{diffRows.map((row) => (
									<li key={row.label}>
										<Text strong className="text-xs">
											{row.label}
										</Text>
										<div className="font-mono text-xs text-text-secondary">
											{row.prevVal} → {row.currVal}
										</div>
									</li>
								))}
							</ul>
						)}
					</Card>
				</>
			) : null}

			<Title level={5} style={{ marginTop: 0, marginBottom: 12 }}>
				结构血缘（摘要）
			</Title>
			<Card size="small">
				<div className="text-sm space-y-2">
					<div>
						<FileOutlined /> 上游:{" "}
						{upstreamMcap ? (
							<Text code className="text-xs">
								MCAP {upstreamMcap}
							</Text>
						) : (
							<Text type="secondary">无 MCAP 关联</Text>
						)}
					</div>
					<div>
						<RobotOutlined /> 下游:{" "}
						<Text>
							{algoCount} 个算法
						</Text>
						<Text type="secondary"> · </Text>
						<SendOutlined />{" "}
						<Text>{deliveryCount} 交付</Text>
					</div>
					{onOpenLineageTab ? (
						<Button
							type="link"
							size="small"
							icon={<LinkOutlined />}
							onClick={onOpenLineageTab}
						>
							在「血缘」Tab 查看完整
						</Button>
					) : null}
				</div>
			</Card>
		</div>
	);
}
