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
import {
	forwardRef,
	type Key,
	useEffect,
	useImperativeHandle,
	useMemo,
	useRef,
	useState,
} from "react";
import {
	type ValidateBackfillAssetsResult,
	validateBackfillAssets,
} from "../../api/batchJobApi";
import type { SearchAssetResult } from "../../api/search";
import { searchApi } from "../../api/search";
import { isCanonicalAssetId } from "../../lib/assetId";
import {
	mergeAssetIds,
	mergePendingBulkPaste,
	parseAssetIdInput,
} from "../../lib/assetIdInput";
import {
	batchAssetLimitError,
	exceedsBatchAssetLimit,
	MAX_BATCH_ASSET_COUNT,
} from "../../lib/batchAssetLimits";
import { withSelectAllColumn } from "../../lib/tableSelection";
import AssetIdLink from "../common/AssetIdLink";

const SEARCH_PAGE_SIZE = 100;

export type AssetPickerHandle = {
	/** Apply uncommitted bulk paste, then return IDs to use for deploy. */
	resolveSelectionForRun: () => { assetIds: string[]; error?: string };
};

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
const AssetPicker = forwardRef<AssetPickerHandle, AssetPickerProps>(
	function AssetPicker(
		{
			selectedIds,
			onSelectionChange,
			placeholder = "搜索资产（输入 asset_id 或名称）",
			maxHeight = 200,
			resetKey,
		},
		ref,
	) {
		const [results, setResults] = useState<SearchAssetResult[]>([]);
		const [loading, setLoading] = useState(false);
		const [query, setQuery] = useState("");
		const [error, setError] = useState<string | null>(null);
		const [bulkPaste, setBulkPaste] = useState("");
		const [assetValidation, setAssetValidation] =
			useState<ValidateBackfillAssetsResult | null>(null);
		const [validationLoading, setValidationLoading] = useState(false);
		const abortRef = useRef<AbortController | null>(null);
		const previousResetKeyRef = useRef(resetKey);
		const selectedIdsRef = useRef(selectedIds);
		const bulkPasteRef = useRef(bulkPaste);

		selectedIdsRef.current = selectedIds;
		bulkPasteRef.current = bulkPaste;

		useImperativeHandle(
			ref,
			() => ({
				resolveSelectionForRun: () => {
					const result = mergePendingBulkPaste(
						selectedIdsRef.current,
						bulkPasteRef.current,
					);
					if (result.error) {
						setError(result.error);
						return result;
					}
					if (parseAssetIdInput(bulkPasteRef.current).length > 0) {
						setBulkPaste("");
						setError(null);
						onSelectionChange(result.assetIds);
					}
					return result;
				},
			}),
			[onSelectionChange],
		);

		useEffect(() => {
			return () => {
				abortRef.current?.abort();
			};
		}, []);

		useEffect(() => {
			if (selectedIds.length === 0) {
				setAssetValidation(null);
				setValidationLoading(false);
				return;
			}
			let cancelled = false;
			const timer = window.setTimeout(() => {
				setValidationLoading(true);
				void validateBackfillAssets(selectedIds)
					.then((result) => {
						if (!cancelled) setAssetValidation(result);
					})
					.catch(() => {
						if (!cancelled) setAssetValidation(null);
					})
					.finally(() => {
						if (!cancelled) setValidationLoading(false);
					});
			}, 300);
			return () => {
				cancelled = true;
				window.clearTimeout(timer);
			};
		}, [selectedIds]);

		const registeredSet = useMemo(
			() => new Set(assetValidation?.registered ?? []),
			[assetValidation],
		);

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
			setAssetValidation(null);
			setValidationLoading(false);
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

		const applyBulkPaste = () => {
			const incoming = parseAssetIdInput(bulkPaste);
			applyAssetIds(incoming);
			setBulkPaste("");
		};

		const handleBulkPasteBlur = () => {
			if (!bulkPaste.trim()) return;
			applyBulkPaste();
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
								color={
									assetValidation
										? registeredSet.has(assetId)
											? "green"
											: "orange"
										: "blue"
								}
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

				{selectedIds.length > 0 ? (
					<Alert
						type={
							assetValidation && assetValidation.unknown.length > 0
								? "warning"
								: "info"
						}
						showIcon
						message={
							assetValidation
								? `目录中已注册 ${assetValidation.registered.length} 个${
										assetValidation.unknown.length > 0
											? `，${assetValidation.unknown.length} 个尚未注册`
											: ""
									}`
								: validationLoading
									? "正在校验资产目录…"
									: "资产目录校验暂不可用"
						}
						description={
							assetValidation ? (
								<Space direction="vertical" size={4}>
									{assetValidation.unknown.length > 0 ? (
										<Typography.Text type="secondary" style={{ fontSize: 12 }}>
											未注册 ID 仍可创建批次；已注册 ID 可跳转资产详情页。
										</Typography.Text>
									) : null}
									{assetValidation.registered.length > 0 ? (
										<Space size={4} wrap>
											{assetValidation.registered
												.filter((id) => isCanonicalAssetId(id))
												.slice(0, 6)
												.map((assetId) => (
													<AssetIdLink key={assetId} id={assetId} />
												))}
											{assetValidation.registered.length > 6 ? (
												<Typography.Text
													type="secondary"
													style={{ fontSize: 12 }}
												>
													+{assetValidation.registered.length - 6}
												</Typography.Text>
											) : null}
										</Space>
									) : null}
								</Space>
							) : undefined
						}
					/>
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

				<Collapse
					size="small"
					items={[
						{
							key: "bulk",
							label: "批量粘贴 asset ID（换行 / 逗号 / 分号分隔）",
							children: (
								<Input.TextArea
									value={bulkPaste}
									onChange={(event) => setBulkPaste(event.target.value)}
									onBlur={handleBulkPasteBlur}
									placeholder={`每行一个 asset_id，或逗号分隔；失焦或点击「运行」时自动导入；单次最多 ${MAX_BATCH_ASSET_COUNT.toLocaleString()} 条`}
									autoSize={{ minRows: 4, maxRows: 10 }}
								/>
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
							: "输入关键字搜索资产，或批量粘贴 asset ID；粘贴后可直接运行"}
					</div>
				)}
			</div>
		);
	},
);

export default AssetPicker;
