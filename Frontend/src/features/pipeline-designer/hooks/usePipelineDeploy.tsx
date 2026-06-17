import { Modal } from "antd";
import { type Dispatch, type RefObject, useCallback, useEffect } from "react";
import {
	BATCH_ASSET_THRESHOLD,
	deployPipelineForAssets,
} from "../../../api/deployPipelineRun";
import {
	listExecutionTargets,
	previewDeploy,
	savePipeline,
} from "../../../api/pipelineApi";
import type { AssetPickerHandle } from "../../../components/pipeline/AssetPicker";
import type { Pipeline } from "../../../components/pipeline/types";
import { batchJobDetailLocationState } from "../../../lib/pipelineNavigation";
import type { DeployDialogState, DesignerAction } from "./designerReducer";

type MessageApi = {
	error: (content: string) => void;
	success: (content: string) => void;
};

interface UsePipelineDeployOptions {
	assetPickerRef: RefObject<AssetPickerHandle | null>;
	buildPipelineJSON: () => Pipeline;
	assertPipelineRunnable: (pipeline: Pipeline, actionLabel: string) => boolean;
	deployDialog: DeployDialogState;
	pipelineName: string;
	selectedAssetIds: string[];
	selectedTargetId: string;
	dispatch: Dispatch<DesignerAction>;
	closeDeployDialog: () => void;
	messageApi: MessageApi;
	navigate: (to: string, options?: { state?: unknown }) => void;
}

export function usePipelineDeploy({
	assetPickerRef,
	buildPipelineJSON,
	assertPipelineRunnable,
	deployDialog,
	pipelineName,
	selectedAssetIds,
	selectedTargetId,
	dispatch,
	closeDeployDialog,
	messageApi,
	navigate,
}: UsePipelineDeployOptions) {
	useEffect(() => {
		let alive = true;
		listExecutionTargets()
			.then((targets) => {
				if (!alive) return;
				dispatch({ type: "deploy/setExecutionTargets", targets });
				const defaultTarget =
					targets.find((target) => target.isDefault) ?? targets[0];
				if (defaultTarget) {
					dispatch({
						type: "deploy/setSelectedTargetId",
						targetId: defaultTarget.id,
					});
				}
			})
			.catch(() => {
				if (!alive) return;
				dispatch({ type: "deploy/setExecutionTargets", targets: [] });
			});
		return () => {
			alive = false;
		};
	}, [dispatch]);

	const handleDeploy = useCallback(async () => {
		const resolved = assetPickerRef.current?.resolveSelectionForRun() ?? {
			assetIds: selectedAssetIds,
		};
		if (resolved.error) {
			messageApi.error(resolved.error);
			return;
		}
		const assetIds = resolved.assetIds;
		const assetCount = assetIds.length;
		if (assetCount >= BATCH_ASSET_THRESHOLD) {
			const confirmed = await new Promise<boolean>((resolve) => {
				Modal.confirm({
					title: "确认创建批量任务",
					content: (
						<div>
							<p>
								将为 {assetCount} 个资产各创建 1 条子任务，共 {assetCount}{" "}
								条执行记录。
							</p>
							<p style={{ marginBottom: 0, color: "#64748b", fontSize: 12 }}>
								提交后可在「执行记录 →
								批量任务」查看进度，并支持暂停或重试失败项。
							</p>
						</div>
					),
					okText: "确认运行",
					cancelText: "取消",
					onOk: () => resolve(true),
					onCancel: () => resolve(false),
				});
			});
			if (!confirmed) return;
		}

		dispatch({
			type: "deploy/setDialog",
			dialog: { deploying: true, done: false },
		});
		try {
			const pipeline = buildPipelineJSON();
			if (!assertPipelineRunnable(pipeline, "运行")) {
				dispatch({
					type: "deploy/setDialog",
					dialog: { deploying: false },
				});
				return;
			}
			const name = deployDialog.name || pipelineName;
			const saved = await savePipeline(name, pipeline);
			const result = await deployPipelineForAssets(saved.id, assetIds, {
				targetId: selectedTargetId,
				batchName: `${name}-${Date.now()}`,
			});
			if (result.mode === "batch") {
				messageApi.success(
					`已创建批量任务，共 ${result.batchJob.totalCount} 个子任务`,
				);
				closeDeployDialog();
				navigate(`/pipeline/batch/${result.batchJob.id}`, {
					state: batchJobDetailLocationState(),
				});
				return;
			}
			dispatch({
				type: "deploy/setDialog",
				dialog: {
					deploying: false,
					done: true,
					result: result.runs[0],
					results: result.runs,
				},
			});
		} catch (err) {
			dispatch({
				type: "deploy/setDialog",
				dialog: {
					deploying: false,
					done: true,
					error: String(err),
				},
			});
		}
	}, [
		assetPickerRef,
		buildPipelineJSON,
		assertPipelineRunnable,
		deployDialog.name,
		pipelineName,
		selectedAssetIds,
		selectedTargetId,
		dispatch,
		closeDeployDialog,
		messageApi,
		navigate,
	]);

	const handlePreviewDeploy = useCallback(async () => {
		dispatch({
			type: "deploy/setDialog",
			dialog: {
				previewLoading: true,
				previewError: undefined,
				previewManifest: undefined,
				mode: "preview",
			},
		});
		try {
			const pipeline = buildPipelineJSON();
			if (!assertPipelineRunnable(pipeline, "预览")) {
				dispatch({
					type: "deploy/setDialog",
					dialog: { previewLoading: false, mode: "edit" },
				});
				return;
			}
			const { manifest } = await previewDeploy(pipeline);
			dispatch({
				type: "deploy/setDialog",
				dialog: { previewLoading: false, previewManifest: manifest },
			});
		} catch (err) {
			dispatch({
				type: "deploy/setDialog",
				dialog: {
					previewLoading: false,
					previewError: String(err),
				},
			});
		}
	}, [buildPipelineJSON, assertPipelineRunnable, dispatch]);

	return { handleDeploy, handlePreviewDeploy };
}
