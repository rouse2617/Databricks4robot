import { Alert, Button, Input, Table } from "antd";
import { useEffect, useRef, useState } from "react";
import { assetsApi } from "../../api/assets";
import type { SearchAssetResult } from "../../api/search";
import { searchApi } from "../../api/search";

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
 *  Wraps no Modal or Collapse — parent controls the container.
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
			let items = res.items;
			if (items.length === 0 && isDirectAssetID(trimmed)) {
				const asset = await lookupDirectAsset(trimmed);
				if (asset) {
					items = [asset];
				}
			}
			if (abortRef.current === controller && !controller.signal.aborted) {
				setResults(items);
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
			<Input.Search
				placeholder={placeholder}
				value={query}
				onChange={(event) => setQuery(event.target.value)}
				onSearch={handleSearch}
				loading={loading}
				size="small"
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
					{query ? "未找到匹配的资产" : "输入关键字搜索资产，不选择则直接部署"}
				</div>
			)}
		</div>
	);
}

function isDirectAssetID(value: string) {
	return /^[A-Za-z0-9][A-Za-z0-9_-]{2,63}$/.test(value);
}

async function lookupDirectAsset(
	assetId: string,
): Promise<SearchAssetResult | null> {
	try {
		return await assetsApi.get(assetId);
	} catch {
		return null;
	}
}
