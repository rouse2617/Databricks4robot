// ─── AssetsPage — Three-column discovery workbench container ───
// Wires all extracted components to the centralized reducer.
// Validates: Requirements R1, R7, R13

import { useEffect, useCallback, useRef, useState } from "react";
import { Typography, message, Modal } from "antd";
import { useNavigate } from "react-router-dom";
import { useAssetsDiscoveryReducer } from "../hooks/assets/useAssetsDiscoveryReducer";
import { useAssetsQuerySync } from "../hooks/assets/useAssetsQuerySync";
import { useSavedViews } from "../hooks/assets/useSavedViews";
import { buildPlaceholderPreviewManifest } from "../hooks/assets/useAssetPreview";
import AssetsSearchBar from "../components/assets/AssetsSearchBar";
import ActiveFilterChipsRow from "../components/assets/ActiveFilterChipsRow";
import AssetsFacetSidebar from "../components/assets/AssetsFacetSidebar";
import AssetsResultsPane from "../components/assets/AssetsResultsPane";
import BulkActionBar from "../components/assets/BulkActionBar";
import AssetQuickPreviewPane from "../components/assets/AssetQuickPreviewPane";
import SavedViewSelector from "../components/assets/SavedViewSelector";
import SaveViewDialog from "../components/assets/SaveViewDialog";
import AddFilterPopover from "../components/assets/AddFilterPopover";
import ExpiringAssetsChip from "../components/assets/ExpiringAssetsChip";
import ColumnsConfigPopover from "../components/assets/ColumnsConfigPopover";
import CreateDeliveryModal from "../components/deliveries/CreateDeliveryModal";
import BatchTagModal from "../components/assets/BatchTagModal";
import BatchDeleteTagModal from "../components/assets/BatchDeleteTagModal";
import ExportModal from "../components/assets/ExportModal";
import type { BatchTagResult } from "../components/assets/BatchTagModal";

const { Title } = Typography;

export default function AssetsPage() {
  const navigate = useNavigate();
  const [state, dispatch] = useAssetsDiscoveryReducer();
  useAssetsQuerySync(state.queryState, state.routerState, dispatch);
  const { views, currentViewId, selectView, saveView, deleteView } = useSavedViews(state, dispatch);
  const [msg, msgCtx] = message.useMessage();
  const containerRef = useRef<HTMLDivElement>(null);
  const [deliveryModalOpen, setDeliveryModalOpen] = useState(false);
  const [batchTagModalOpen, setBatchTagModalOpen] = useState(false);
  const [batchDeleteTagModalOpen, setBatchDeleteTagModalOpen] = useState(false);
  const [exportModalOpen, setExportModalOpen] = useState(false);
  const [batchTagResult, setBatchTagResult] = useState<BatchTagResult | null>(null);

  // Responsive: hide facet/preview on narrow screens
  // Use lazy initializer to avoid window access during SSR
  const [isNarrow, setIsNarrow] = useState(() =>
    typeof window !== "undefined" ? window.innerWidth < 1024 : false
  );
  useEffect(() => {
    const handler = () => setIsNarrow(window.innerWidth < 1024);
    window.addEventListener("resize", handler);
    return () => window.removeEventListener("resize", handler);
  }, []);

  // Scroll to top on page change
  useEffect(() => {
    containerRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
  }, [state.queryState.page]);

  // ── Keyboard handler: Up/Down arrows change activePreviewId, / focuses search, Escape closes dropdowns, Space toggles selection ──
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement).tagName;
      const isInInput = tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";

      // `/` focuses search bar (when not already in an input)
      if (e.key === "/" && !isInInput) {
        e.preventDefault();
        const searchInput = document.querySelector<HTMLInputElement>(
          '[data-testid="assets-search-input"]',
        );
        searchInput?.focus();
        return;
      }

      // `Escape` closes suggestion dropdown / popovers
      if (e.key === "Escape") {
        dispatch({ type: "TOGGLE_SUGGESTIONS", payload: { open: false } });
        dispatch({ type: "TOGGLE_COLUMNS_POPOVER", payload: { open: false } });
        dispatch({ type: "TOGGLE_ADD_FILTER", payload: { open: false } });
        return;
      }

      // Don't intercept arrow/space if user is in an input/textarea
      if (isInInput) return;

      // `Space` toggles row selection for the active preview row
      if (e.key === " ") {
        const activeId = state.previewState.activeAssetId;
        if (activeId) {
          e.preventDefault();
          dispatch({ type: "TOGGLE_ROW_SELECTION", payload: { id: activeId } });
        }
        return;
      }

      // `ArrowUp`/`ArrowDown` change activePreviewId
      if (e.key !== "ArrowUp" && e.key !== "ArrowDown") return;

      const items = state.resultsState.items;
      if (items.length === 0) return;

      const currentId = state.previewState.activeAssetId;
      const currentIdx = currentId
        ? items.findIndex((a) => a.asset_id === currentId)
        : -1;

      let nextIdx: number;
      if (e.key === "ArrowDown") {
        nextIdx = currentIdx < items.length - 1 ? currentIdx + 1 : currentIdx;
      } else {
        nextIdx = currentIdx > 0 ? currentIdx - 1 : 0;
      }

      const nextAsset = items[nextIdx];
      if (nextAsset && nextAsset.asset_id !== currentId) {
        e.preventDefault();
        dispatch({
          type: "SET_ACTIVE_PREVIEW_ASSET",
          payload: { assetId: nextAsset.asset_id },
        });
      }
    },
    [state.resultsState.items, state.previewState.activeAssetId, dispatch],
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

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

      {/* Search bar + Saved Views + Add Filter */}
      <div style={{ marginBottom: 8, display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
        <div style={{ flex: 1, minWidth: 200 }}>
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
        <SavedViewSelector
          views={views}
          currentViewId={currentViewId}
          onSelectView={selectView}
          onDeleteView={deleteView}
          onOpenSaveDialog={() => dispatch({ type: "TOGGLE_SAVE_DIALOG", payload: { open: true } })}
        />
        <AddFilterPopover
          onAddFilter={(chip) =>
            dispatch({ type: "ADD_FILTER_CHIP", payload: { chip } })
          }
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
      </div>

      {/* Save View Dialog */}
      <SaveViewDialog
        open={state.savedViewState.saveDialogOpen}
        onSave={(name) => saveView(name)}
        onCancel={() => dispatch({ type: "TOGGLE_SAVE_DIALOG", payload: { open: false } })}
      />

      {/* Active filter chips */}
      <div style={{ marginBottom: 8 }}>
        <ActiveFilterChipsRow
          chips={state.queryState.activeFilters}
          onRemoveChip={(id) =>
            dispatch({ type: "REMOVE_FILTER_CHIP", payload: { id } })
          }
          onClearAll={() => dispatch({ type: "CLEAR_ALL_FILTERS" })}
        />
      </div>

      {/* Three-column layout */}
      <div style={{ display: "flex", gap: 12, alignItems: "flex-start" }}>
        {/* Left: Facet Sidebar */}
        {!isNarrow && (
        <div style={{ width: 220, flexShrink: 0 }}>
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 8 }}>
            <span style={{ fontSize: 13, fontWeight: 600, color: "#1E293B" }}>筛选</span>
            {state.queryState.activeFilters.length > 0 && (
              <a
                onClick={() => {
                  dispatch({ type: "CLEAR_ALL_FILTERS" });
                  dispatch({ type: "FACET_RANGE_DRAFT", payload: { field: "duration_ms", min: undefined, max: undefined } });
                }}
                style={{ fontSize: 12, color: "#2563EB", cursor: "pointer" }}
              >
                重置
              </a>
            )}
          </div>
          <AssetsFacetSidebar
            activeFilters={state.queryState.activeFilters}
            expandedGroups={state.facetUiState.expandedGroups}
            rangeDrafts={state.facetUiState.rangeDrafts}
            dateDrafts={state.facetUiState.dateDrafts}
            aggregations={state.resultsState.aggregations}
            onToggleFacet={(field, value) =>
              dispatch({ type: "FACET_TOGGLE", payload: { field, value } })
            }
            onApplyRange={(field, min, max) =>
              dispatch({ type: "FACET_RANGE_APPLY", payload: { field, min, max } })
            }
            onApplyDate={(field, start, end) =>
              dispatch({ type: "FACET_DATE_APPLY", payload: { field, start, end } })
            }
            onRangeDraftChange={(field, min, max) =>
              dispatch({ type: "FACET_RANGE_DRAFT", payload: { field, min, max } })
            }
            onDateDraftChange={(field, start, end) =>
              dispatch({ type: "FACET_DATE_DRAFT", payload: { field, start, end } })
            }
            onToggleGroup={(group) =>
              dispatch({ type: "FACET_GROUP_TOGGLE", payload: { group } })
            }
          />
        </div>
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
            onSelectAllFiltered={() => dispatch({ type: "SELECT_ALL_FILTERED" })}
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
            onRowClick={(assetId) =>
              dispatch({
                type: "SET_ACTIVE_PREVIEW_ASSET",
                payload: { assetId },
              })
            }
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
                  dispatch({ type: "TOGGLE_COLUMNS_POPOVER", payload: { open } })
                }
                onColumnsChange={(columns) =>
                  dispatch({ type: "SET_SELECTED_COLUMNS", payload: { columns } })
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
            onOpenDetail={(assetId) => navigate(`/assets/${assetId}`)}
            onFindSimilar={() => {}}
            onRetry={() => {
              const id = state.previewState.activeAssetId;
              if (id) {
                dispatch({ type: "SET_ACTIVE_PREVIEW_ASSET", payload: { assetId: null } });
                // Re-trigger by setting the asset id again on next tick
                setTimeout(() => {
                  dispatch({ type: "SET_ACTIVE_PREVIEW_ASSET", payload: { assetId: id } });
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
          filter: state.queryState.activeFilters.length > 0
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
          dispatch({ type: "SET_PAGE", payload: { page: state.queryState.page } });
        }}
        onCancel={() => setBatchTagResult(null)}
        okText="确定"
        cancelButtonProps={{ style: { display: "none" } }}
      >
        {batchTagResult && (
          <div style={{ fontSize: 15, lineHeight: 2 }}>
            <div>✅ 成功：<strong>{batchTagResult.success}</strong> 个</div>
            <div>⏭️ 跳过：<strong>{batchTagResult.skipped}</strong> 个</div>
            <div>❌ 失败：<strong>{batchTagResult.failed}</strong> 个</div>
          </div>
        )}
      </Modal>
    </div>
  );
}
