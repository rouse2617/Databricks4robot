import {
	Alert,
	Button,
	Collapse,
	Input,
	Space,
	Table,
	Tag,
	Typography,
} from "antd";
import { type Key, useEffect, useRef, useState } from "react";
import type { SearchAssetResult } from "../../api/search";
import { searchApi } from "../../api/search";
import { mergeAssetIds, parseAssetIdInput } from "../../lib/assetIdInput";
import {
	batchAssetLimitError,
	exceedsBatchAssetLimit,
	MAX_BATCH_ASSET_COUNT,
} from "../../lib/batchAssetLimits";
import { withSelectAllColumn } from "../../lib/tableSelection";

const SEARCH_PAGE_SIZE = 100;

interface AssetPickerProps {
	/** Currently selected asset IDs (controlled) */
	selectedIds: string[];
	/** Called when selection changes */
	onSelectionChange: (ids: string[]) => void;
	/** Placeholder text for the search input */
	placeholder?: string;
	/** Table max height for scroll */
	maxHeight?: number;
	/** Changes to this value reset transient search text/results. */
	resetKey?: string | number;
}

/** Shared asset search + multi-select table.
 *  Supports search hits, manual asset ID entry, and bulk paste (no registry required).
 */
export default function AssetPicker({
	selectedIds,
	onSelectionChange,
	placeholder = "搜索资产（输入 asset_id 或名称）",
	maxHeight = 200,
	resetKey,
}: AssetPickerProps) {
	const [results, setResults] = useState<SearchAssetResult[]>([]);
	const [loading, setLoading] = useState(false);
	const [query, setQuery] = useState("");
	const [error, setError] = useState<string | null>(null);
	const [bulkPaste, setBulkPaste] = useState("");
	const abortRef = useRef<AbortController | null>(null);
	const previousResetKeyRef = useRef(resetKey);

	useEffect(() => {
		return () => {
			abortRef.current?.abort();
		};
	}, []);

	useEffect(() => {
		if (previousResetKeyRef.current === resetKey) return;
		previousResetKeyRef.current = resetKey;
		abortRef.current?.abort();
		abortRef.current = null;
		setQuery("");
		setBulkPaste("");
		setResults([]);
		setError(null);
		setLoading(false);
	}, [resetKey]);

	const isAbortError = (err: unknown) =>
		(err instanceof DOMException && err.name === "AbortError") ||
		(typeof err === "object" &&
			err !== null &&
			("name" in err || "code" in err) &&
			((err as { name?: string }).name === "AbortError" ||
				(err as { code?: string }).code === "ERR_CANCELED"));

	const removeSelected = (assetId: string) => {
		onSelectionChange(selectedIds.filter((id) => id !== assetId));
	};

	const applyAssetIds = (incoming: string[]) => {
		if (incoming.length === 0) return;
		const merged = mergeAssetIds(selectedIds, incoming);
		if (exceedsBatchAssetLimit(merged.length)) {
			setError(batchAssetLimitError(merged.length));
			return;
		}
		setError(null);
		onSelectionChange(merged);
	};

	const addManualAssetIds = (raw: string) => {
		const incoming = parseAssetIdInput(raw);
		applyAssetIds(incoming);
		setQuery("");
		setResults([]);
	};

	const handleBulkPasteApply = () => {
		const incoming = parseAssetIdInput(bulkPaste);
		applyAssetIds(incoming);
		setBulkPaste("");
	};

	const handleTableSelectionChange = (keys: Key[]) => {
		const pageIds = new Set(results.map((row) => row.asset_id));
		const offPageSelected = selectedIds.filter((id) => !pageIds.has(id));
		const merged = mergeAssetIds(offPageSelected, keys as string[]);
		if (exceedsBatchAssetLimit(merged.length)) {
			setError(batchAssetLimitError(merged.length));
			return;
		}
		setError(null);
		onSelectionChange(merged);
	};

	const handleSearch = async (value: string) => {
		const trimmed = value.trim();
		if (!trimmed) {
			abortRef.current?.abort();
			abortRef.current = null;
			setQuery("");
			setResults([]);
			setError(null);
			setLoading(false);
			return;
		}
		abortRef.current?.abort();
		const controller = new AbortController();
		abortRef.current = controller;
		setQuery(trimmed);
		setLoading(true);
		setError(null);
		try {
			const res = await searchApi.searchAssets(
				{ q: trimmed, page_size: SEARCH_PAGE_SIZE },
				controller.signal,
			);
			if (abortRef.current === controller && !controller.signal.aborted) {
				setResults(res.items);
			}
		} catch (err) {
			if (!isAbortError(err) && abortRef.current === controller) {
				setError("搜索资产失败，请重试");
			}
		} finally {
			if (abortRef.current === controller) {
				setLoading(false);
			}
		}
	};

	return (
		<div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
			{selectedIds.length > 0 ? (
				<div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
					<Typography.Text type="secondary" style={{ fontSize: 12 }}>
						已选 {selectedIds.length} 个
					</Typography.Text>
					{selectedIds.slice(0, 12).map((assetId) => (
						<Tag
							key={assetId}
							closable
							color="blue"
							onClose={() => removeSelected(assetId)}
						>
							{assetId}
						</Tag>
					))}
					{selectedIds.length > 12 ? (
						<Tag>+{selectedIds.length - 12}</Tag>
					) : null}
				</div>
			) : null}

			<Input.Search
				placeholder={placeholder}
				value={query}
				onChange={(event) => setQuery(event.target.value)}
				onSearch={handleSearch}
				loading={loading}
				size="small"
			/>

			<Space size={8} wrap>
				<Button
					size="small"
					disabled={!query.trim()}
					onClick={() => addManualAssetIds(query)}
				>
					{query.trim() ? `添加「${query.trim()}」为资产 ID` : "添加为资产 ID"}
				</Button>
				<Button
					size="small"
					type="link"
					disabled={selectedIds.length === 0}
					onClick={() => onSelectionChange([])}
				>
					清空已选
				</Button>
			</Space>

			<Collapse
				size="small"
				items={[
					{
						key: "bulk",
						label: "批量粘贴 asset ID（换行 / 逗号 / 分号分隔）",
						children: (
							<Space direction="vertical" style={{ width: "100%" }} size={8}>
								<Input.TextArea
									value={bulkPaste}
									onChange={(event) => setBulkPaste(event.target.value)}
									placeholder={`每行一个 asset_id，或逗号分隔；单次批量最多 ${MAX_BATCH_ASSET_COUNT.toLocaleString()} 条`}
									autoSize={{ minRows: 4, maxRows: 10 }}
								/>
								<Button
									size="small"
									type="primary"
									disabled={!bulkPaste.trim()}
									onClick={handleBulkPasteApply}
								>
									导入到已选列表
								</Button>
							</Space>
						),
					},
				]}
			/>

			{error ? (
				<Alert
					type="error"
					showIcon
					message={error}
					action={
						<Button size="small" danger onClick={() => handleSearch(query)}>
							重试
						</Button>
					}
				/>
			) : results.length > 0 ? (
				<Table
					rowKey="asset_id"
					dataSource={results}
					size="small"
					pagination={false}
					scroll={{ y: maxHeight }}
					rowSelection={withSelectAllColumn<SearchAssetResult>({
						type: "checkbox",
						selectedRowKeys: selectedIds,
						onChange: handleTableSelectionChange,
					})}
					columns={[
						{ title: "Asset ID", dataIndex: "asset_id", width: 120 },
						{ title: "类型", dataIndex: "asset_type", width: 70 },
						{ title: "状态", dataIndex: "lifecycle_state", width: 70 },
						{
							title: "存储路径",
							dataIndex: "storage_uri",
							width: 180,
							render: (v: string | undefined) =>
								v ? (
									<span
										style={{
											fontSize: 11,
											fontFamily: '"SF Mono", monospace',
											color: "#64748b",
										}}
									>
										{v.length > 36 ? `${v.slice(0, 36)}\u2026` : v}
									</span>
								) : (
									<span style={{ fontSize: 11, color: "#aaa" }}>—</span>
								),
						},
					]}
				/>
			) : (
				<div
					style={{
						color: "#999",
						textAlign: "center",
						padding: 12,
						fontSize: 12,
					}}
				>
					{query
						? "未找到匹配的资产，仍可点击「添加为资产 ID」或批量粘贴"
						: "输入关键字搜索资产，或批量粘贴 asset ID；不选择则直接部署"}
				</div>
			)}
		</div>
	);
}
