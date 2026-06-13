import { Alert, Button, Input, Space, Tag, Table } from "antd";
import { useEffect, useRef, useState } from "react";
import type { SearchAssetResult } from "../../api/search";
import { searchApi } from "../../api/search";
import { mergeAssetIds, parseAssetIdInput } from "../../lib/assetIdInput";

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
 *  Supports search hits and manual asset ID entry (no registry required).
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

	const addManualAssetIds = (raw: string) => {
		const incoming = parseAssetIdInput(raw);
		if (incoming.length === 0) return;
		onSelectionChange(mergeAssetIds(selectedIds, incoming));
		setQuery("");
		setResults([]);
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
				{ q: trimmed, page_size: 50 },
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
					{selectedIds.map((assetId) => (
						<Tag
							key={assetId}
							closable
							color="blue"
							onClose={() => removeSelected(assetId)}
						>
							{assetId}
						</Tag>
					))}
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
					{query.trim()
						? `添加「${query.trim()}」为资产 ID`
						: "添加为资产 ID"}
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
					rowSelection={{
						type: "checkbox",
						selectedRowKeys: selectedIds,
						onChange: (keys) => onSelectionChange(keys as string[]),
					}}
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
						? "未找到匹配的资产，仍可点击「添加为资产 ID」直接用于批量任务"
						: "输入关键字搜索资产，或手动添加 asset ID；不选择则直接部署"}
				</div>
			)}
		</div>
	);
}
