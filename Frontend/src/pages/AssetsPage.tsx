// ─── AssetsPage — Three-column discovery workbench container ───
// Wires all extracted components to the centralized reducer.
// Validates: Requirements R1, R7, R13

import { FilterOutlined } from "@ant-design/icons";
import {
	Badge,
	Button,
	Drawer,
	Modal,
	message,
	Popover,
	Typography,
} from "antd";
import { useEffect, useRef, useState } from "react";
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
import SearchSyncStatusAlert from "../components/assets/SearchSyncStatusAlert";
import TagAdvancedFilter from "../components/assets/TagAdvancedFilter";
import CreateDeliveryModal from "../components/deliveries/CreateDeliveryModal";
import { buildPlaceholderPreviewManifest } from "../hooks/assets/useAssetPreview";
import { useAssetsDiscoveryReducer } from "../hooks/assets/useAssetsDiscoveryReducer";
import { useAssetsHotkeys } from "../hooks/assets/useAssetsHotkeys";
import { useAssetsQuerySync } from "../hooks/assets/useAssetsQuerySync";
import { serializeQueryStateToUrl } from "../lib/assets/assetsDiscoveryUrl";
import { navigateToAssetDetail } from "../lib/assets/assetWorkbenchNavigation";

const { Title } = Typography;

export default function AssetsPage() {
	const navigate = useNavigate();
	const [state, dispatch] = useAssetsDiscoveryReducer();
	useAssetsQuerySync(
		state.queryState,
		state.routerState,
		state.previewState.activeAssetId,
		dispatch,
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
	const [tagAdvancedOpen, setTagAdvancedOpen] = useState(false);
	const [filterDrawerOpen, setFilterDrawerOpen] = useState(false);

	// Responsive: hide facet/preview on narrow screens
	// Use lazy initializer to avoid window access during SSR
	const [isNarrow, setIsNarrow] = useState(() =>
		typeof window !== "undefined" ? window.innerWidth < 1024 : false,
	);
	useEffect(() => {
		const handler = () => setIsNarrow(window.innerWidth < 1024);
		window.addEventListener("resize", handler);
		return () => window.removeEventListener("resize", handler);
	}, []);

	// Scroll to top on page change
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

	// ── Derived state ──
	const selectedCount = state.selectionState.selectedIds.size;
	const previewAsset = state.previewState.summary;
	const previewManifest = previewAsset
		? buildPlaceholderPreviewManifest(previewAsset)
		: null;

	return (
		<div ref={containerRef}>
			{msgCtx}

			{/* Page title */}
			<Title level={4} style={{ margin: "0 0 12px 0" }}>
				资产管理
			</Title>

			<SearchSyncStatusAlert />

			{/* Search bar + Saved Views + Add Filter */}
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
				<Popover
					content={
						<TagAdvancedFilter
							onApply={(chips) => {
								for (const chip of chips) {
									dispatch({ type: "ADD_FILTER_CHIP", payload: { chip } });
								}
							}}
							onClose={() => setTagAdvancedOpen(false)}
						/>
					}
					title={null}
					trigger="click"
					open={tagAdvancedOpen}
					onOpenChange={setTagAdvancedOpen}
					placement="bottomLeft"
				>
					<Button icon={<FilterOutlined />}>Tag 高级筛选</Button>
				</Popover>
				<ExpiringAssetsChip
					activeFilters={state.queryState.activeFilters}
					onAddFilter={(chip) =>
						dispatch({ type: "ADD_FILTER_CHIP", payload: { chip } })
					}
					onRemoveFilter={(id) =>
						dispatch({ type: "REMOVE_FILTER_CHIP", payload: { id } })
					}
				/>
			</div>

			<div style={{ marginBottom: 8 }}>
				<QuickFiltersRow
					activeFilters={state.queryState.activeFilters}
					onAddFilter={(chip) =>
						dispatch({ type: "ADD_FILTER_CHIP", payload: { chip } })
					}
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

			{/* Active filter chips */}
			<div style={{ marginBottom: 8 }}>
				<ActiveFilterChipsRow
					chips={state.queryState.activeFilters}
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
			</div>

			{/* Main layout */}
			<div style={{ display: "flex", gap: 12, alignItems: "flex-start" }}>
				{/* Left: Facet sidebar (always visible on wide screens) */}
				{!isNarrow && (
					<aside
						style={{
							width: 240,
							flexShrink: 0,
							position: "sticky",
							top: 12,
							maxHeight: "calc(100vh - 24px)",
							overflowY: "auto",
							padding: 12,
							border: "1px solid #E5E7EB",
							borderRadius: 8,
							background: "#fff",
						}}
					>
						<div
							style={{
								display: "flex",
								justifyContent: "space-between",
								alignItems: "center",
								marginBottom: 8,
							}}
						>
							<span style={{ fontSize: 13, fontWeight: 600, color: "#1E293B" }}>
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
					</aside>
				)}

				{/* Center: Bulk Action Bar + Columns Config + Results */}
				<div style={{ flex: 1, minWidth: 0 }}>
					<BulkActionBar
						selectedCount={selectedCount}
						selectionMode={state.selectionState.mode}
						totalFiltered={state.resultsState.total}
						onCreateDelivery={() => setDeliveryModalOpen(true)}
						onRunAlgo={() => msg.warning("批量触发算法功能开发中，敬请期待")}
						onBatchTag={() => setBatchTagModalOpen(true)}
						onBatchDeleteTag={() => setBatchDeleteTagModalOpen(true)}
						onExportIds={() => setExportModalOpen(true)}
						onSelectAllFiltered={() =>
							dispatch({ type: "SELECT_ALL_FILTERED" })
						}
						onClearSelection={() => dispatch({ type: "CLEAR_SELECTION" })}
						disabledRunAlgo
					/>
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
						onPageChange={(page) =>
							dispatch({ type: "SET_PAGE", payload: { page } })
						}
						onSelectRow={(id) =>
							dispatch({ type: "TOGGLE_ROW_SELECTION", payload: { id } })
						}
						onRowClick={(assetId) => {
							// Narrow layout hides the quick-preview column — navigate so the click has a visible result.
							if (isNarrow) {
								const sp = serializeQueryStateToUrl(state.queryState, assetId);
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

				{/* Right: Quick Preview Pane */}
				{!isNarrow && (
					<div
						style={{
							width: state.previewState.collapsed ? 36 : 320,
							flexShrink: 0,
							transition: "width 0.2s ease",
						}}
					>
						<AssetQuickPreviewPane
							activeAssetId={state.previewState.activeAssetId}
							fetchStatus={state.previewState.fetchStatus}
							asset={previewAsset}
							previewManifest={previewManifest}
							collapsed={state.previewState.collapsed}
							onCollapse={() => dispatch({ type: "PREVIEW_COLLAPSE_TOGGLE" })}
							onOpenDetail={(assetId) => {
								const sp = serializeQueryStateToUrl(state.queryState, assetId);
								const qs = sp.toString();
								navigateToAssetDetail(navigate, assetId, {
									state: { assetsReturnTo: qs ? `/assets?${qs}` : "/assets" },
								});
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
								const id = state.previewState.activeAssetId;
								if (id) {
									dispatch({
										type: "SET_ACTIVE_PREVIEW_ASSET",
										payload: { assetId: null },
									});
									// Re-trigger by setting the asset id again on next tick
									setTimeout(() => {
										dispatch({
											type: "SET_ACTIVE_PREVIEW_ASSET",
											payload: { assetId: id },
										});
									}, 0);
								}
							}}
						/>
					</div>
				)}
			</div>

			<CreateDeliveryModal
				open={deliveryModalOpen}
				assetIds={Array.from(state.selectionState.selectedIds)}
				onClose={() => setDeliveryModalOpen(false)}
				onSuccess={(deliveryId) => {
					setDeliveryModalOpen(false);
					dispatch({ type: "CLEAR_SELECTION" });
					navigate(`/deliveries/${deliveryId}`);
				}}
			/>

			<BatchTagModal
				open={batchTagModalOpen}
				assetIds={Array.from(state.selectionState.selectedIds)}
				onClose={() => setBatchTagModalOpen(false)}
				onComplete={(result) => {
					setBatchTagModalOpen(false);
					setBatchTagResult(result);
				}}
			/>

			<BatchDeleteTagModal
				open={batchDeleteTagModalOpen}
				assetIds={Array.from(state.selectionState.selectedIds)}
				onClose={() => setBatchDeleteTagModalOpen(false)}
				onComplete={(result) => {
					setBatchDeleteTagModalOpen(false);
					setBatchTagResult(result);
				}}
			/>

			<ExportModal
				open={exportModalOpen}
				currentPageItems={state.resultsState.items}
				totalFiltered={state.resultsState.total}
				queryParams={{
					filter:
						state.queryState.activeFilters.length > 0
							? state.queryState.activeFilters.map((chip) => {
									const val = Array.isArray(chip.value)
										? `[${chip.value.map((v) => `"${v}"`).join(",")}]`
										: chip.value;
									return `${chip.field}:${chip.op}:${val}`;
								})
							: undefined,
					sort_by: state.queryState.sort,
				}}
				onClose={() => setExportModalOpen(false)}
			/>

			{/* Result summary modal (Task 4.6) */}
			<Modal
				title="操作结果"
				open={batchTagResult !== null}
				onOk={() => {
					setBatchTagResult(null);
					dispatch({ type: "CLEAR_SELECTION" });
					dispatch({
						type: "SET_PAGE",
						payload: { page: state.queryState.page },
					});
				}}
				onCancel={() => setBatchTagResult(null)}
				okText="确定"
				cancelButtonProps={{ style: { display: "none" } }}
			>
				{batchTagResult && (
					<div style={{ fontSize: 15, lineHeight: 2 }}>
						<div>
							✅ 成功：<strong>{batchTagResult.success}</strong> 个
						</div>
						<div>
							⏭️ 跳过：<strong>{batchTagResult.skipped}</strong> 个
						</div>
						<div>
							❌ 失败：<strong>{batchTagResult.failed}</strong> 个
						</div>
					</div>
				)}
			</Modal>
		</div>
	);
}
