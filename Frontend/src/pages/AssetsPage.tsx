// ─── AssetsPage — Three-column discovery workbench container ───
// Wires all extracted components to the centralized reducer.
// Validates: Requirements R1, R7, R13

import {
	AppstoreOutlined,
	FilterOutlined,
	TableOutlined,
	UnorderedListOutlined,
} from "@ant-design/icons";
import {
	Alert,
	Badge,
	Button,
	Drawer,
	Modal,
	message,
	Segmented,
	Typography,
} from "antd";
import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import ActiveFilterChipsRow from "../components/assets/ActiveFilterChipsRow";
import AddFilterPopover from "../components/assets/AddFilterPopover";
import AssetQuickPreviewPane from "../components/assets/AssetQuickPreviewPane";
import AssetsFacetSidebar from "../components/assets/AssetsFacetSidebar";
import AssetsResultsPane from "../components/assets/AssetsResultsPane";
import AssetsSearchBar from "../components/assets/AssetsSearchBar";
import BatchDeleteTagModal from "../components/assets/BatchDeleteTagModal";
import type { BatchTagResult } from "../components/assets/BatchTagModal";
import BatchTagModal from "../components/assets/BatchTagModal";
import BulkActionBar from "../components/assets/BulkActionBar";
import ColumnsConfigPopover from "../components/assets/ColumnsConfigPopover";
import ExpiringAssetsChip from "../components/assets/ExpiringAssetsChip";
import ExportModal from "../components/assets/ExportModal";
import QuickFiltersRow from "../components/assets/QuickFiltersRow";

const CreateDeliveryModal = lazy(
	() => import("../components/deliveries/CreateDeliveryModal"),
);

import { queryApi } from "../api/query";
import {
	buildSelectAllIdsQueryRequest,
	SELECT_ALL_MAX,
	useAssetsDiscoveryReducer,
} from "../hooks/assets/useAssetsDiscoveryReducer";
import { useAssetsHotkeys } from "../hooks/assets/useAssetsHotkeys";
import { useAssetsQuerySync } from "../hooks/assets/useAssetsQuerySync";
import type { ViewMode } from "../lib/assets/assetsDiscoveryTypes";
import { serializeQueryStateToUrl } from "../lib/assets/assetsDiscoveryUrl";
import { navigateToAssetDetail } from "../lib/assets/assetWorkbenchNavigation";

const { Title } = Typography;

function translateResultWarning(warning: string): string {
	const normalized = warning.toLowerCase();
	if (
		normalized.includes("elasticsearch unavailable") &&
		normalized.includes("postgres-only")
	) {
		return "搜索索引暂不可用，已改用数据库查询；当前筛选仍已执行，但结果可能更慢。";
	}
	if (
		normalized.includes("elasticsearch unavailable") &&
		normalized.includes("postgres count")
	) {
		return "搜索索引暂不可用，数量统计已改用数据库结果。";
	}
	return warning;
}

export default function AssetsPage() {
	const navigate = useNavigate();
	const [state, dispatch] = useAssetsDiscoveryReducer();
	useAssetsQuerySync(
		state.queryState,
		state.routerState,
		state.previewState.activeAssetId,
		state.previewState.activeSourceId,
		state.previewState.activeTopic,
		dispatch,
		state.previewState.activeTimeSec,
		state.previewState.ds,
		state.previewState.dsParams,
	);
	const [msg, msgCtx] = message.useMessage();
	const containerRef = useRef<HTMLDivElement>(null);
	const [deliveryModalOpen, setDeliveryModalOpen] = useState(false);
	const [batchTagModalOpen, setBatchTagModalOpen] = useState(false);
	const [batchDeleteTagModalOpen, setBatchDeleteTagModalOpen] = useState(false);
	const [exportModalOpen, setExportModalOpen] = useState(false);
	const [batchTagResult, setBatchTagResult] = useState<BatchTagResult | null>(
		null,
	);
	const [filterDrawerOpen, setFilterDrawerOpen] = useState(false);

	const [isNarrow, setIsNarrow] = useState(() =>
		typeof window !== "undefined" ? window.innerWidth < 1024 : false,
	);
	const [isUltraWide, setIsUltraWide] = useState(() =>
		typeof window !== "undefined" ? window.innerWidth > 1800 : false,
	);
	useEffect(() => {
		const handler = () => {
			setIsNarrow(window.innerWidth < 1024);
			setIsUltraWide(window.innerWidth > 1800);
		};
		window.addEventListener("resize", handler);
		return () => window.removeEventListener("resize", handler);
	}, []);

	useEffect(() => {
		containerRef.current?.scrollIntoView({
			behavior: "smooth",
			block: "start",
		});
	}, []);

	useAssetsHotkeys({
		containerRef,
		items: state.resultsState.items,
		activeAssetId: state.previewState.activeAssetId,
		onSetActiveAsset: (assetId) => {
			dispatch({ type: "SET_ACTIVE_PREVIEW_ASSET", payload: { assetId } });
		},
		onToggleActiveSelection: () => {
			if (!state.previewState.activeAssetId) return;
			dispatch({
				type: "TOGGLE_ROW_SELECTION",
				payload: { id: state.previewState.activeAssetId },
			});
		},
		onCloseOverlays: () => {
			dispatch({ type: "TOGGLE_SUGGESTIONS", payload: { open: false } });
			dispatch({ type: "TOGGLE_COLUMNS_POPOVER", payload: { open: false } });
			dispatch({ type: "TOGGLE_ADD_FILTER", payload: { open: false } });
		},
	});

	const selectedCount = state.selectionState.selectedIds.size;
	const previewAsset = state.previewState.summary;
	const previewManifest = state.previewState.manifest;
	const resultWarnings = state.resultsState.warnings ?? [];
	const visibleResultWarnings = resultWarnings.map(translateResultWarning);
	const hasSearchFallbackWarning = resultWarnings.some((w) =>
		w.toLowerCase().includes("elasticsearch unavailable"),
	);
	const hasAlgoStatusWarning = resultWarnings.some((w) =>
		w.includes("algo_status"),
	);
	const selectedAssetIds = Array.from(state.selectionState.selectedIds);
	const canRunPipelineForSelection =
		state.selectionState.mode === "explicit_rows" &&
		selectedAssetIds.length > 0;
	const runPipelineForSelectedAssets = () => {
		if (!canRunPipelineForSelection) {
			msg.warning("请先逐行选择要处理的资产，再运行 Pipeline");
			return;
		}
		navigate(
			`/pipeline?tab=pipelines&asset_ids=${encodeURIComponent(selectedAssetIds.join(","))}`,
		);
	};

	return (
		<div ref={containerRef} style={{ width: "100%", padding: "0 16px" }}>
			{msgCtx}

			<Title level={4} style={{ margin: "0 0 12px 0" }}>
				资产管理
			</Title>

			<div
				style={{
					marginBottom: 8,
					display: "flex",
					alignItems: "center",
					gap: 8,
					flexWrap: "wrap",
				}}
			>
				<div style={{ flex: "0 1 480px", minWidth: 240 }}>
					<AssetsSearchBar
						searchMode={state.queryState.searchMode}
						draftText={state.searchUiState.draftText}
						committedQueryText={state.queryState.queryText}
						onDraftChange={(text) =>
							dispatch({ type: "SET_SEARCH_DRAFT", payload: { text } })
						}
						onCommitQuery={(text, tokens) =>
							dispatch({ type: "COMMIT_QUERY_TEXT", payload: { text, tokens } })
						}
						onModeChange={(mode) =>
							dispatch({ type: "SET_SEARCH_MODE", payload: { mode } })
						}
					/>
				</div>
				{isNarrow && (
					<Badge
						count={state.queryState.activeFilters.length}
						size="small"
						offset={[-4, 4]}
					>
						<Button
							icon={<FilterOutlined />}
							type={
								state.queryState.activeFilters.length > 0
									? "primary"
									: "default"
							}
							ghost={state.queryState.activeFilters.length > 0}
							onClick={() => setFilterDrawerOpen(true)}
							data-testid="open-filters-drawer"
						>
							筛选
						</Button>
					</Badge>
				)}
				<AddFilterPopover
					onAddFilter={(chip) =>
						dispatch({ type: "ADD_FILTER_CHIP", payload: { chip } })
					}
				/>
				<div
					style={{
						width: 1,
						height: 16,
						background: "#e5e7eb",
						margin: "0 4px",
					}}
				/>
				<ExpiringAssetsChip
					activeFilters={state.queryState.activeFilters}
					onAddFilter={(chip) =>
						dispatch({ type: "ADD_FILTER_CHIP", payload: { chip } })
					}
					onRemoveFilter={(id) =>
						dispatch({ type: "REMOVE_FILTER_CHIP", payload: { id } })
					}
				/>
				<QuickFiltersRow
					activeFilters={state.queryState.activeFilters}
					onAddFilter={(chip) => {
						dispatch({ type: "ADD_FILTER_CHIP", payload: { chip } });
						// Auto-expand algorithm tab when algo_status is clicked
						if (
							chip.field === "algo_status" &&
							!state.facetUiState.expandedGroups.includes("algorithm")
						) {
							dispatch({
								type: "FACET_GROUP_TOGGLE",
								payload: { group: "algorithm" },
							});
						}
					}}
					onRemoveFilter={(id) =>
						dispatch({ type: "REMOVE_FILTER_CHIP", payload: { id } })
					}
				/>
			</div>

			<Drawer
				title={
					<div
						style={{
							display: "flex",
							justifyContent: "space-between",
							alignItems: "center",
						}}
					>
						<span>筛选</span>
						{state.queryState.activeFilters.length > 0 && (
							<button
								type="button"
								className="link-like-button"
								onClick={() => {
									dispatch({ type: "CLEAR_ALL_FILTERS" });
									dispatch({
										type: "FACET_RANGE_DRAFT",
										payload: {
											field: "duration_ms",
											min: undefined,
											max: undefined,
										},
									});
								}}
								style={{
									fontSize: 12,
									color: "#2563EB",
									cursor: "pointer",
									marginRight: 24,
								}}
							>
								重置
							</button>
						)}
					</div>
				}
				placement="right"
				width={400}
				open={filterDrawerOpen}
				onClose={() => setFilterDrawerOpen(false)}
				destroyOnHidden={false}
				styles={{ body: { overscrollBehavior: "contain" } }}
			>
				<AssetsFacetSidebar
					layout="vertical"
					activeFilters={state.queryState.activeFilters}
					expandedGroups={state.facetUiState.expandedGroups}
					rangeDrafts={state.facetUiState.rangeDrafts}
					dateDrafts={state.facetUiState.dateDrafts}
					aggregations={state.resultsState.aggregations}
					onToggleFacet={(field, value) =>
						dispatch({ type: "FACET_TOGGLE", payload: { field, value } })
					}
					onApplyRange={(field, min, max) =>
						dispatch({
							type: "FACET_RANGE_APPLY",
							payload: { field, min, max },
						})
					}
					onApplyDate={(field, start, end) =>
						dispatch({
							type: "FACET_DATE_APPLY",
							payload: { field, start, end },
						})
					}
					onRangeDraftChange={(field, min, max) =>
						dispatch({
							type: "FACET_RANGE_DRAFT",
							payload: { field, min, max },
						})
					}
					onDateDraftChange={(field, start, end) =>
						dispatch({
							type: "FACET_DATE_DRAFT",
							payload: { field, start, end },
						})
					}
					onToggleGroup={(group) =>
						dispatch({ type: "FACET_GROUP_TOGGLE", payload: { group } })
					}
				/>
			</Drawer>

			<ActiveFilterChipsRow
				chips={state.queryState.activeFilters}
				filteredTotal={state.resultsState.total}
				totalApprox={state.resultsState.totalApprox}
				onRemoveChip={(id) =>
					dispatch({ type: "REMOVE_FILTER_CHIP", payload: { id } })
				}
				onClearField={(field) => {
					for (const chip of state.queryState.activeFilters.filter(
						(item) => item.field === field,
					)) {
						dispatch({
							type: "REMOVE_FILTER_CHIP",
							payload: { id: chip.id },
						});
					}
				}}
				onClearAll={() => dispatch({ type: "CLEAR_ALL_FILTERS" })}
			/>

			{resultWarnings.length > 0 && (
				<Alert
					type="warning"
					showIcon
					style={{ marginBottom: 8 }}
					message={
						hasSearchFallbackWarning
							? "搜索已降级为数据库查询"
							: hasAlgoStatusWarning
								? "⚠️ algo_status 筛选仅在当前页生效"
								: "部分筛选可能未完全生效"
					}
					description={visibleResultWarnings.join("；")}
				/>
			)}

			{!isNarrow && (
				<div
					style={{
						marginBottom: 8,
						padding: "10px 12px",
						border: "1px solid #E5E7EB",
						borderRadius: 10,
						background: "#fff",
						boxShadow: "0 1px 2px rgba(15, 23, 42, 0.04)",
					}}
				>
					<div
						style={{
							display: "flex",
							justifyContent: "space-between",
							alignItems: "center",
							marginBottom: 6,
							gap: 8,
							flexWrap: "wrap",
						}}
					>
						<span style={{ fontSize: 12, fontWeight: 600, color: "#1E293B" }}>
							筛选
						</span>
						{state.queryState.activeFilters.length > 0 && (
							<button
								type="button"
								className="link-like-button"
								onClick={() => {
									dispatch({ type: "CLEAR_ALL_FILTERS" });
									dispatch({
										type: "FACET_RANGE_DRAFT",
										payload: {
											field: "duration_ms",
											min: undefined,
											max: undefined,
										},
									});
								}}
								style={{ fontSize: 12, color: "#2563EB", cursor: "pointer" }}
							>
								重置
							</button>
						)}
					</div>
					<AssetsFacetSidebar
						layout="horizontal"
						compact
						activeFilters={state.queryState.activeFilters}
						expandedGroups={state.facetUiState.expandedGroups}
						rangeDrafts={state.facetUiState.rangeDrafts}
						dateDrafts={state.facetUiState.dateDrafts}
						aggregations={state.resultsState.aggregations}
						onToggleFacet={(field, value) =>
							dispatch({ type: "FACET_TOGGLE", payload: { field, value } })
						}
						onApplyRange={(field, min, max) =>
							dispatch({
								type: "FACET_RANGE_APPLY",
								payload: { field, min, max },
							})
						}
						onApplyDate={(field, start, end) =>
							dispatch({
								type: "FACET_DATE_APPLY",
								payload: { field, start, end },
							})
						}
						onRangeDraftChange={(field, min, max) =>
							dispatch({
								type: "FACET_RANGE_DRAFT",
								payload: { field, min, max },
							})
						}
						onDateDraftChange={(field, start, end) =>
							dispatch({
								type: "FACET_DATE_DRAFT",
								payload: { field, start, end },
							})
						}
						onToggleGroup={(group) =>
							dispatch({ type: "FACET_GROUP_TOGGLE", payload: { group } })
						}
					/>
				</div>
			)}

			<div style={{ display: "flex", gap: 16, alignItems: "flex-start" }}>
				<div style={{ flex: 1, minWidth: 0 }}>
					<BulkActionBar
						selectedCount={selectedCount}
						selectionMode={state.selectionState.mode}
						totalFiltered={state.resultsState.total}
						onCreateDelivery={() => setDeliveryModalOpen(true)}
						onRunPipeline={runPipelineForSelectedAssets}
						onBatchTag={() => setBatchTagModalOpen(true)}
						onBatchDeleteTag={() => setBatchDeleteTagModalOpen(true)}
						onExportIds={() => setExportModalOpen(true)}
						onSelectAllFiltered={async () => {
							// CYB-3231: resolve the matching asset_ids (capped) and select
							// them explicitly, so the count and every bulk action reflect the
							// full filtered set — not just the rows checked by hand.
							try {
								const data = await queryApi.run(
									buildSelectAllIdsQueryRequest(
										state.queryState,
										SELECT_ALL_MAX,
									),
								);
								const ids = (data.items ?? [])
									.map((a) => a.asset_id)
									.filter((id): id is string => Boolean(id));
								dispatch({ type: "SET_SELECTED_IDS", payload: { ids } });
								const total = state.resultsState.total ?? 0;
								if (total > ids.length) {
									void msg.warning(
										`已选择前 ${ids.length} 条（上限 ${SELECT_ALL_MAX}），筛选结果共 ${total} 条，其余未选`,
									);
								} else {
									void msg.success(`已选择全部 ${ids.length} 条`);
								}
							} catch {
								void msg.error("全选失败，请重试");
							}
						}}
						onClearSelection={() => dispatch({ type: "CLEAR_SELECTION" })}
						disabledRunPipeline={!canRunPipelineForSelection}
					/>
					<div
						style={{
							display: "flex",
							justifyContent: "flex-end",
							marginBottom: 8,
						}}
					>
						<Segmented
							value={state.queryState.viewMode}
							options={[
								{
									label: (
										<span>
											<AppstoreOutlined /> 卡片
										</span>
									),
									value: "card",
								},
								{
									label: (
										<span>
											<TableOutlined /> 表格
										</span>
									),
									value: "table",
								},
								{
									label: (
										<span>
											<UnorderedListOutlined /> 紧凑
										</span>
									),
									value: "compact",
								},
							]}
							onChange={(value) =>
								dispatch({
									type: "SET_VIEW_MODE",
									payload: { mode: value as ViewMode },
								})
							}
						/>
					</div>
					<AssetsResultsPane
						items={state.resultsState.items}
						total={state.resultsState.total}
						totalApprox={state.resultsState.totalApprox}
						fetchStatus={state.resultsState.fetchStatus}
						error={state.resultsState.error}
						activeFilterCount={state.queryState.activeFilters.length}
						hasActiveQueryText={state.queryState.queryText.trim().length > 0}
						sort={state.queryState.sort}
						page={state.queryState.page}
						pageSize={state.queryState.pageSize}
						viewMode={state.queryState.viewMode}
						selectedColumns={state.queryState.selectedColumns}
						selectedIds={state.selectionState.selectedIds}
						activePreviewId={state.previewState.activeAssetId}
						onSortChange={(sort) =>
							dispatch({ type: "SET_SORT", payload: { sort } })
						}
						onPageChange={(page, pageSize) => {
							if (pageSize !== state.queryState.pageSize) {
								dispatch({ type: "SET_PAGE_SIZE", payload: { pageSize } });
								return;
							}
							dispatch({ type: "SET_PAGE", payload: { page } });
						}}
						onSelectRow={(id) =>
							dispatch({ type: "TOGGLE_ROW_SELECTION", payload: { id } })
						}
						onRowClick={(assetId) => {
							if (isNarrow) {
								const sp = serializeQueryStateToUrl(
									state.queryState,
									assetId,
									state.previewState.activeSourceId,
									state.previewState.activeTopic,
								);
								const qs = sp.toString();
								navigateToAssetDetail(navigate, assetId, {
									state: { assetsReturnTo: qs ? `/assets?${qs}` : "/assets" },
								});
								return;
							}
							dispatch({
								type: "SET_ACTIVE_PREVIEW_ASSET",
								payload: { assetId },
							});
						}}
						onMcapClick={(mcapFileId) => {
							navigate(
								`/mcap-files?mcap_file_id=${encodeURIComponent(mcapFileId)}`,
							);
						}}
						onClearFilters={() => dispatch({ type: "CLEAR_ALL_FILTERS" })}
						onExport={() => setExportModalOpen(true)}
						onRetry={() =>
							dispatch({
								type: "SET_PAGE",
								payload: { page: state.queryState.page },
							})
						}
						columnsConfigSlot={
							<ColumnsConfigPopover
								selectedColumns={state.queryState.selectedColumns}
								open={state.layoutState.columnsPopoverOpen}
								onOpenChange={(open) =>
									dispatch({
										type: "TOGGLE_COLUMNS_POPOVER",
										payload: { open },
									})
								}
								onColumnsChange={(columns) =>
									dispatch({
										type: "SET_SELECTED_COLUMNS",
										payload: { columns },
									})
								}
							/>
						}
					/>
				</div>

				{!isNarrow && (
					<div
						style={{
							width: isUltraWide || !state.previewState.collapsed ? 320 : 36,
							flexShrink: 0,
							alignSelf: "flex-start",
							position: "sticky",
							top: 12,
							maxHeight: "calc(100vh - 24px)",
							transition: "width 0.2s ease",
						}}
					>
						<AssetQuickPreviewPane
							activeAssetId={state.previewState.activeAssetId}
							fetchStatus={state.previewState.fetchStatus}
							asset={previewAsset}
							previewManifest={previewManifest}
							collapsed={isUltraWide ? false : state.previewState.collapsed}
							onCollapse={() =>
								!isUltraWide && dispatch({ type: "PREVIEW_COLLAPSE_TOGGLE" })
							}
							onOpenDetail={(assetId) => {
								const sp = serializeQueryStateToUrl(
									state.queryState,
									assetId,
									state.previewState.activeSourceId,
									state.previewState.activeTopic,
								);
								const qs = sp.toString();
								navigateToAssetDetail(navigate, assetId, {
									state: { assetsReturnTo: qs ? `/assets?${qs}` : "/assets" },
								});
							}}
							onMcapClick={(mcapFileId) => {
								navigate(
									`/mcap-files?mcap_file_id=${encodeURIComponent(mcapFileId)}`,
								);
							}}
							onFindSimilar={(assetId) => {
								dispatch({
									type: "SET_SEARCH_MODE",
									payload: { mode: "similar" },
								});
								dispatch({
									type: "COMMIT_QUERY_TEXT",
									payload: { text: assetId, tokens: [] },
								});
							}}
							onRetry={() => {
								dispatch({ type: "PREVIEW_RETRY" });
							}}
							previewSourceId={state.previewState.activeSourceId}
							onSourceChange={(sourceId) => {
								dispatch({ type: "SET_PREVIEW_SOURCE", payload: { sourceId } });
							}}
							previewTopic={state.previewState.activeTopic}
							onTopicChange={(topic) => {
								dispatch({ type: "SET_PREVIEW_TOPIC", payload: { topic } });
							}}
							previewTimeSec={state.previewState.activeTimeSec}
							onPreviewTimeChange={(timeSec) => {
								dispatch({ type: "SET_PREVIEW_TIME", payload: { timeSec } });
							}}
						/>
					</div>
				)}
			</div>

			{deliveryModalOpen ? (
				<Suspense fallback={null}>
					<CreateDeliveryModal
						open={deliveryModalOpen}
						assetIds={selectedAssetIds}
						onClose={() => setDeliveryModalOpen(false)}
						onSuccess={async (deliveryId) => {
							setDeliveryModalOpen(false);
							dispatch({ type: "CLEAR_SELECTION" });
							await msg.success(`已创建交付 ${deliveryId}`);
						}}
					/>
				</Suspense>
			) : null}
			<BatchTagModal
				open={batchTagModalOpen}
				assetIds={selectedAssetIds}
				onClose={() => setBatchTagModalOpen(false)}
				onComplete={(result) => {
					setBatchTagModalOpen(false);
					setBatchTagResult(result);
					dispatch({ type: "CLEAR_SELECTION" });
				}}
			/>
			<BatchDeleteTagModal
				open={batchDeleteTagModalOpen}
				assetIds={selectedAssetIds}
				onClose={() => setBatchDeleteTagModalOpen(false)}
				onComplete={(result) => {
					setBatchDeleteTagModalOpen(false);
					msg.success(
						`已删除标签：成功 ${result.success}，失败 ${result.failed}`,
					);
					dispatch({ type: "CLEAR_SELECTION" });
				}}
			/>
			<ExportModal
				open={exportModalOpen}
				currentPageItems={state.resultsState.items}
				totalFiltered={state.resultsState.total}
				queryParams={{
					sort_by: state.queryState.sort,
					page: state.queryState.page,
					page_size: state.queryState.pageSize,
				}}
				onClose={() => setExportModalOpen(false)}
			/>
			<Modal
				open={!!batchTagResult}
				title="批量打标结果"
				onCancel={() => setBatchTagResult(null)}
				onOk={() => setBatchTagResult(null)}
			>
				{batchTagResult && (
					<div style={{ display: "grid", gap: 8 }}>
						<div>成功：{batchTagResult.success}</div>
						<div>跳过：{batchTagResult.skipped}</div>
						<div>失败：{batchTagResult.failed}</div>
					</div>
				)}
			</Modal>
		</div>
	);
}
