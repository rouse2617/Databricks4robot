// ─── BatchTagModal — Batch add/update tags for selected assets ───
// Validates: Requirements REQ-3.1.1, REQ-3.1.2, REQ-3.1.3, REQ-3.1.5, REQ-3.1.6

import {
	Input,
	Modal,
	message,
	Progress,
	Radio,
	Select,
	Space,
	Typography,
} from "antd";
import { useCallback, useEffect, useState } from "react";
import { assetsApi } from "../../api/assets";
import { type TagRegistryItem, tagRegistryApi } from "../../api/tagRegistry";

const { Text } = Typography;

export type ConflictStrategy = "overwrite" | "skip";

export interface BatchTagResult {
	success: number;
	skipped: number;
	failed: number;
}

export interface BatchTagModalProps {
	open: boolean;
	assetIds: string[];
	onClose: () => void;
	onComplete: (result: BatchTagResult) => void;
}

/** Run async tasks with concurrency limit */
async function runWithConcurrency<T>(
	tasks: (() => Promise<T>)[],
	limit: number,
	onProgress: () => void,
): Promise<T[]> {
	const results: T[] = new Array(tasks.length);
	let idx = 0;

	async function worker() {
		while (idx < tasks.length) {
			const i = idx++;
			results[i] = await tasks[i]();
			onProgress();
		}
	}

	await Promise.all(
		Array.from({ length: Math.min(limit, tasks.length) }, () => worker()),
	);
	return results;
}

export default function BatchTagModal({
	open,
	assetIds,
	onClose,
	onComplete,
}: BatchTagModalProps) {
	const [tags, setTags] = useState<TagRegistryItem[]>([]);
	const [loading, setLoading] = useState(false);
	const [selectedKey, setSelectedKey] = useState<string | undefined>();
	const [tagValue, setTagValue] = useState<string>("");
	const [strategy, setStrategy] = useState<ConflictStrategy>("overwrite");
	const [executing, setExecuting] = useState(false);
	const [progress, setProgress] = useState(0);
	const [msg, msgCtx] = message.useMessage();

	const selectedTag = tags.find((t) => t.key === selectedKey);

	// Fetch tag registry on open
	useEffect(() => {
		if (!open) return;
		setLoading(true);
		tagRegistryApi
			.list()
			.then(setTags)
			.catch(() => msg.error("获取标签注册表失败"))
			.finally(() => setLoading(false));
	}, [open, msg.error]);

	// Reset form when modal opens
	useEffect(() => {
		if (open) {
			setSelectedKey(undefined);
			setTagValue("");
			setStrategy("overwrite");
			setProgress(0);
			setExecuting(false);
		}
	}, [open]);

	const canSubmit = !!selectedKey && tagValue.trim().length > 0 && !executing;

	const handleExecute = useCallback(async () => {
		if (!selectedKey || !tagValue.trim()) return;

		setExecuting(true);
		setProgress(0);

		let success = 0;
		let skipped = 0;
		let failed = 0;
		let completed = 0;

		const tasks = assetIds.map((assetId) => async () => {
			try {
				// Skip strategy: check if tag already exists
				if (strategy === "skip") {
					const asset = await assetsApi.get(assetId);
					if (
						asset.tags &&
						asset.tags[selectedKey] !== undefined &&
						asset.tags[selectedKey] !== ""
					) {
						skipped++;
						return;
					}
				}
				await assetsApi.upsertTag(assetId, {
					key: selectedKey,
					value: tagValue.trim(),
				});
				success++;
			} catch {
				failed++;
			}
		});

		await runWithConcurrency(tasks, 5, () => {
			completed++;
			setProgress(Math.round((completed / assetIds.length) * 100));
		});

		setExecuting(false);
		onComplete({ success, skipped, failed });
	}, [selectedKey, tagValue, strategy, assetIds, onComplete]);

	return (
		<Modal
			title="批量打标签"
			open={open}
			onOk={handleExecute}
			onCancel={onClose}
			okText={executing ? "执行中..." : "确认执行"}
			cancelText="取消"
			okButtonProps={{ disabled: !canSubmit }}
			closable={!executing}
			maskClosable={!executing}
			destroyOnHidden
		>
			{msgCtx}

			<div
				style={{
					background: "#e6f7ff",
					border: "1px solid #91d5ff",
					borderRadius: 4,
					padding: "8px 12px",
					marginBottom: 16,
					fontSize: 13,
				}}
			>
				将影响 <Text strong>{assetIds.length}</Text> 个资产
			</div>

			<Space direction="vertical" style={{ width: "100%" }} size={16}>
				{/* Tag Key Select */}
				<div>
					<Text style={{ display: "block", marginBottom: 4 }}>标签 Key</Text>
					<Select
						style={{ width: "100%" }}
						placeholder="选择标签 Key"
						loading={loading}
						value={selectedKey}
						onChange={(v) => {
							setSelectedKey(v);
							setTagValue("");
						}}
						options={tags.map((t) => ({
							label: `${t.key} (${t.type})`,
							value: t.key,
						}))}
					/>
				</div>

				{/* Tag Value Input — enum=Select, string=Input */}
				{selectedTag && (
					<div>
						<Text style={{ display: "block", marginBottom: 4 }}>标签值</Text>
						{selectedTag.type === "enum" &&
						selectedTag.values &&
						selectedTag.values.length > 0 ? (
							<Select
								style={{ width: "100%" }}
								placeholder="选择标签值"
								value={tagValue || undefined}
								onChange={setTagValue}
								options={selectedTag.values.map((v) => ({
									label: v,
									value: v,
								}))}
							/>
						) : (
							<Input
								placeholder="输入标签值"
								value={tagValue}
								onChange={(e) => setTagValue(e.target.value)}
								maxLength={selectedTag.max_length}
								showCount={!!selectedTag.max_length}
							/>
						)}
					</div>
				)}

				{/* Conflict Strategy */}
				<div>
					<Text style={{ display: "block", marginBottom: 4 }}>冲突策略</Text>
					<Radio.Group
						value={strategy}
						onChange={(e) => setStrategy(e.target.value)}
					>
						<Radio value="overwrite">覆盖已有值</Radio>
						<Radio value="skip">跳过已有值</Radio>
					</Radio.Group>
				</div>

				{/* Progress Bar */}
				{executing && <Progress percent={progress} status="active" />}
			</Space>
		</Modal>
	);
}
