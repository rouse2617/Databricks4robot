import { StarFilled } from "@ant-design/icons";
import {
	Alert,
	Button,
	Drawer,
	List,
	Modal,
	Skeleton,
	Tag,
	Timeline,
	Typography,
} from "antd";
import { useCallback, useEffect, useState } from "react";
import {
	getPipelineDiff,
	listPipelineVersions,
	type PipelineDiff,
	type PipelineTemplate,
} from "../../api/pipelineApi";

const { Text } = Typography;

export interface VersionHistoryDrawerProps {
	open: boolean;
	pipelineName: string;
	templateId: string;
	activeVersion?: number;
	onClose: () => void;
	onSetActive?: (version: number) => void;
}

export function VersionHistoryDrawer({
	open,
	pipelineName,
	templateId,
	activeVersion,
	onClose,
	onSetActive,
}: VersionHistoryDrawerProps) {
	const [versions, setVersions] = useState<PipelineTemplate[]>([]);
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [diffOpen, setDiffOpen] = useState(false);
	const [diffLoading, setDiffLoading] = useState(false);
	const [diffError, setDiffError] = useState<string | null>(null);
	const [diffResult, setDiffResult] = useState<PipelineDiff | null>(null);
	const [diffLabel, setDiffLabel] = useState("");

	const fetch = useCallback(async () => {
		if (!open) return;
		setLoading(true);
		setError(null);
		try {
			const items = await listPipelineVersions(templateId);
			setVersions(items);
		} catch (err) {
			const detail = err instanceof Error ? err.message : String(err);
			setError(detail);
			setVersions([]);
		} finally {
			setLoading(false);
		}
	}, [open, templateId]);

	useEffect(() => {
		fetch();
	}, [fetch]);

	const showDiff = async (from: PipelineTemplate, to: PipelineTemplate) => {
		setDiffOpen(true);
		setDiffLoading(true);
		setDiffError(null);
		setDiffResult(null);
		setDiffLabel(`v${from.version} → v${to.version}`);
		try {
			const diff = await getPipelineDiff(from.id, to.id);
			setDiffResult(diff);
		} catch (err) {
			setDiffError(err instanceof Error ? err.message : String(err));
		} finally {
			setDiffLoading(false);
		}
	};

	const latestVersion = versions.length > 0 ? versions[0]?.version : 0;
	const effectiveActive = activeVersion ?? latestVersion;

	return (
		<>
			<Drawer
				title={`${pipelineName} 版本历史`}
				open={open}
				onClose={onClose}
				width={400}
				data-testid="version-history-drawer"
			>
				<Alert
					type="info"
					showIcon
					style={{ marginBottom: 12 }}
					message="版本与正式版说明"
					description={
						<ul style={{ margin: 0, paddingLeft: 18, fontSize: 12 }}>
							<li>
								<strong>活跃版本</strong>：未指定版本时，「运行」默认使用的 dev
								模板版本。
							</li>
							<li>
								<strong>发布到正式版（prod）</strong>：生成只读 prod
								快照，用于生产运行；与活跃版本相互独立。
							</li>
							<li>prod 模板不可编辑或删除，可在设计页以只读模式查看 DAG。</li>
						</ul>
					}
				/>
				{loading ? (
					<Skeleton active paragraph={{ rows: 4 }} />
				) : error ? (
					<Alert type="error" message="加载失败" description={error} showIcon />
				) : versions.length === 0 ? (
					<Alert type="info" message="暂无版本记录" showIcon />
				) : (
					<Timeline
						items={versions.map((v, index) => {
							const isLatest = v.version === latestVersion;
							const isActive = v.version === effectiveActive;
							const older = versions[index + 1];
							return {
								key: v.id,
								color: isActive ? "blue" : "gray",
								dot: isActive ? (
									<StarFilled style={{ fontSize: 14 }} />
								) : undefined,
								children: (
									<div className="version-timeline-entry">
										<div className="version-timeline-entry__header">
											<Tag color={isActive ? "blue" : "default"}>
												v{v.version}
											</Tag>
											{isLatest ? (
												<Tag color="green" style={{ fontSize: 11 }}>
													最新
												</Tag>
											) : null}
											{isActive && !isLatest ? (
												<Tag color="orange" style={{ fontSize: 11 }}>
													活跃
												</Tag>
											) : null}
										</div>
										<div className="version-timeline-entry__meta">
											<Text type="secondary" style={{ fontSize: 12 }}>
												{v.nodeCount} 个步骤
											</Text>
											<Text
												type="secondary"
												style={{ fontSize: 12, marginLeft: 8 }}
											>
												{new Date(v.createdAt).toLocaleString()}
											</Text>
										</div>
										<div style={{ display: "flex", gap: 8, marginTop: 4 }}>
											{onSetActive && !isActive ? (
												<Button
													type="link"
													size="small"
													style={{ padding: 0, fontSize: 12 }}
													onClick={() => onSetActive(v.version)}
												>
													设为活跃版本
												</Button>
											) : null}
											{older ? (
												<Button
													type="link"
													size="small"
													style={{ padding: 0, fontSize: 12 }}
													data-testid={`version-diff-${v.version}`}
													onClick={() => void showDiff(older, v)}
												>
													与 v{older.version} 对比
												</Button>
											) : null}
										</div>
									</div>
								),
							};
						})}
					/>
				)}
			</Drawer>
			<Modal
				title={`版本差异 ${diffLabel}`}
				open={diffOpen}
				onCancel={() => setDiffOpen(false)}
				footer={null}
				width={560}
				data-testid="version-diff-modal"
			>
				{diffLoading ? (
					<Skeleton active paragraph={{ rows: 4 }} />
				) : diffError ? (
					<Alert
						type="error"
						message="对比失败"
						description={diffError}
						showIcon
					/>
				) : diffResult ? (
					<div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
						<DiffSection
							title="新增节点"
							items={diffResult.added_nodes.map((node) => node.id)}
							color="green"
						/>
						<DiffSection
							title="删除节点"
							items={diffResult.removed_nodes.map((node) => node.id)}
							color="red"
						/>
						<DiffSection
							title="修改节点"
							items={diffResult.modified_nodes.map((node) => node.id)}
							color="orange"
						/>
						<DiffSection
							title="新增边"
							items={diffResult.added_edges.map(
								(edge) => `${edge.source} → ${edge.target}`,
							)}
							color="blue"
						/>
						<DiffSection
							title="删除边"
							items={diffResult.removed_edges.map(
								(edge) => `${edge.source} → ${edge.target}`,
							)}
							color="default"
						/>
					</div>
				) : null}
			</Modal>
		</>
	);
}

function DiffSection({
	title,
	items,
	color,
}: {
	title: string;
	items: string[];
	color: string;
}) {
	if (items.length === 0) return null;
	return (
		<div>
			<Text strong>
				{title} ({items.length})
			</Text>
			<List
				size="small"
				dataSource={items}
				renderItem={(item) => (
					<List.Item style={{ padding: "4px 0" }}>
						<Tag color={color}>{item}</Tag>
					</List.Item>
				)}
			/>
		</div>
	);
}
