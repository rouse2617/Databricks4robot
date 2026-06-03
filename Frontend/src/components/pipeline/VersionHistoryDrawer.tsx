import { StarFilled } from "@ant-design/icons";
import {
	Alert,
	Button,
	Drawer,
	Skeleton,
	Tag,
	Timeline,
	Typography,
} from "antd";
import { useCallback, useEffect, useState } from "react";
import {
	listPipelineVersions,
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

	const latestVersion = versions.length > 0 ? versions[0]?.version : 0;
	const effectiveActive = activeVersion ?? latestVersion;

	return (
		<Drawer
			title={`${pipelineName} 版本历史`}
			open={open}
			onClose={onClose}
			width={400}
			data-testid="version-history-drawer"
		>
			{loading ? (
				<Skeleton active paragraph={{ rows: 4 }} />
			) : error ? (
				<Alert type="error" message="加载失败" description={error} showIcon />
			) : versions.length === 0 ? (
				<Alert type="info" message="暂无版本记录" showIcon />
			) : (
				<Timeline
					items={versions.map((v) => {
						const isLatest = v.version === latestVersion;
						const isActive = v.version === effectiveActive;
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
									{onSetActive && !isActive ? (
										<Button
											type="link"
											size="small"
											style={{
												padding: 0,
												fontSize: 12,
												marginTop: 4,
											}}
											onClick={() => onSetActive(v.version)}
										>
											设为活跃版本
										</Button>
									) : null}
								</div>
							),
						};
					})}
				/>
			)}
		</Drawer>
	);
}
