import type { Deployment, ExecutionTarget, PipelineTemplate } from "../../../api/pipelineApi";
import type { Asset } from "../../../api/types";
import type { CanvasMenuState } from "../model/canvas-model";

export type DeployMode = "edit" | "preview";

export interface DeployDialogState {
	open: boolean;
	deploying: boolean;
	done: boolean;
	name: string;
	mode: DeployMode;
	result?: Deployment;
	results?: Deployment[];
	error?: string;
	previewManifest?: string;
	previewLoading?: boolean;
	previewError?: string;
}

export interface DesignerState {
	canvas: {
		pipelineName: string;
		selectedNodeId: string | null;
		editingNodeId: string | null;
		contextMenu: CanvasMenuState;
		jsonOutput: string | null;
		importModalOpen: boolean;
		importText: string;
	};
	template: {
		templateLoading: boolean;
		templateVersions: PipelineTemplate[];
		selectedTemplateVersionId: string | null;
		loadedTemplateScope: string | null;
	};
	deploy: {
		deployDialog: DeployDialogState;
		selectedAssetIds: string[];
		assetPickerResetKey: number;
		executionTargets: ExecutionTarget[];
		selectedTargetId: string;
	};
	assetContext: {
		selectedNodeAsset: Asset | null;
		selectedNodeAssetLoading: boolean;
		selectedNodeAssetError: string | null;
	};
}

const initialDeployDialog: DeployDialogState = {
	open: false,
	deploying: false,
	done: false,
	name: "",
	mode: "edit",
};

export const initialDesignerState: DesignerState = {
	canvas: {
		pipelineName: "my-pipeline",
		selectedNodeId: null,
		editingNodeId: null,
		contextMenu: { open: false, x: 0, y: 0, nodeId: null },
		jsonOutput: null,
		importModalOpen: false,
		importText: "",
	},
	template: {
		templateLoading: false,
		templateVersions: [],
		selectedTemplateVersionId: null,
		loadedTemplateScope: null,
	},
	deploy: {
		deployDialog: initialDeployDialog,
		selectedAssetIds: [],
		assetPickerResetKey: 0,
		executionTargets: [],
		selectedTargetId: "default",
	},
	assetContext: {
		selectedNodeAsset: null,
		selectedNodeAssetLoading: false,
		selectedNodeAssetError: null,
	},
};

export type DesignerAction =
	| { type: "canvas/setPipelineName"; name: string }
	| { type: "canvas/selectNode"; nodeId: string | null }
	| { type: "canvas/setEditingNodeId"; nodeId: string | null }
	| { type: "canvas/setContextMenu"; menu: CanvasMenuState }
	| { type: "canvas/closeContextMenu" }
	| { type: "canvas/setJsonOutput"; json: string | null }
	| { type: "canvas/setImportModalOpen"; open: boolean }
	| { type: "canvas/setImportText"; text: string }
	| { type: "canvas/resetImport" }
	| {
			type: "template/setLoading";
			loading: boolean;
	  }
	| {
			type: "template/setVersions";
			versions: PipelineTemplate[];
			selectedVersionId?: string | null;
			loadedScope?: string | null;
	  }
	| { type: "template/clearVersions" }
	| { type: "template/setSelectedVersionId"; versionId: string | null }
	| { type: "template/setLoadedScope"; scope: string | null }
	| { type: "deploy/setDialog"; dialog: Partial<DeployDialogState> }
	| { type: "deploy/resetDialog" }
	| { type: "deploy/openDialog"; name: string }
	| { type: "deploy/setSelectedAssetIds"; assetIds: string[] }
	| { type: "deploy/bumpAssetPickerResetKey" }
	| { type: "deploy/setExecutionTargets"; targets: ExecutionTarget[] }
	| { type: "deploy/setSelectedTargetId"; targetId: string }
	| { type: "assetContext/setLoading"; loading: boolean }
	| { type: "assetContext/setAsset"; asset: Asset | null }
	| { type: "assetContext/setError"; error: string | null }
	| {
			type: "assetContext/reset";
	  };

export function designerReducer(
	state: DesignerState,
	action: DesignerAction,
): DesignerState {
	switch (action.type) {
		case "canvas/setPipelineName":
			return {
				...state,
				canvas: { ...state.canvas, pipelineName: action.name },
			};
		case "canvas/selectNode":
			return {
				...state,
				canvas: { ...state.canvas, selectedNodeId: action.nodeId },
			};
		case "canvas/setEditingNodeId":
			return {
				...state,
				canvas: { ...state.canvas, editingNodeId: action.nodeId },
			};
		case "canvas/setContextMenu":
			return {
				...state,
				canvas: { ...state.canvas, contextMenu: action.menu },
			};
		case "canvas/closeContextMenu":
			return {
				...state,
				canvas: {
					...state.canvas,
					contextMenu: { ...state.canvas.contextMenu, open: false },
				},
			};
		case "canvas/setJsonOutput":
			return {
				...state,
				canvas: { ...state.canvas, jsonOutput: action.json },
			};
		case "canvas/setImportModalOpen":
			return {
				...state,
				canvas: { ...state.canvas, importModalOpen: action.open },
			};
		case "canvas/setImportText":
			return {
				...state,
				canvas: { ...state.canvas, importText: action.text },
			};
		case "canvas/resetImport":
			return {
				...state,
				canvas: {
					...state.canvas,
					importModalOpen: false,
					importText: "",
				},
			};
		case "template/setLoading":
			return {
				...state,
				template: { ...state.template, templateLoading: action.loading },
			};
		case "template/setVersions":
			return {
				...state,
				template: {
					...state.template,
					templateVersions: action.versions,
					selectedTemplateVersionId:
						action.selectedVersionId !== undefined
							? action.selectedVersionId
							: state.template.selectedTemplateVersionId,
					loadedTemplateScope:
						action.loadedScope !== undefined
							? action.loadedScope
							: state.template.loadedTemplateScope,
				},
			};
		case "template/clearVersions":
			if (
				state.template.templateVersions.length === 0 &&
				state.template.selectedTemplateVersionId === null &&
				state.template.loadedTemplateScope === null
			) {
				return state;
			}
			return {
				...state,
				template: {
					...state.template,
					templateVersions: [],
					selectedTemplateVersionId: null,
					loadedTemplateScope: null,
				},
			};
		case "template/setSelectedVersionId":
			return {
				...state,
				template: {
					...state.template,
					selectedTemplateVersionId: action.versionId,
				},
			};
		case "template/setLoadedScope":
			return {
				...state,
				template: {
					...state.template,
					loadedTemplateScope: action.scope,
				},
			};
		case "deploy/setDialog":
			return {
				...state,
				deploy: {
					...state.deploy,
					deployDialog: {
						...state.deploy.deployDialog,
						...action.dialog,
					},
				},
			};
		case "deploy/resetDialog":
			return {
				...state,
				deploy: {
					...state.deploy,
					deployDialog: initialDeployDialog,
					assetPickerResetKey: state.deploy.assetPickerResetKey + 1,
				},
			};
		case "deploy/openDialog":
			return {
				...state,
				deploy: {
					...state.deploy,
					deployDialog: {
						open: true,
						deploying: false,
						done: false,
						mode: "edit",
						name: action.name,
					},
					assetPickerResetKey: state.deploy.assetPickerResetKey + 1,
				},
			};
		case "deploy/setSelectedAssetIds":
			return {
				...state,
				deploy: {
					...state.deploy,
					selectedAssetIds: action.assetIds,
				},
			};
		case "deploy/bumpAssetPickerResetKey":
			return {
				...state,
				deploy: {
					...state.deploy,
					assetPickerResetKey: state.deploy.assetPickerResetKey + 1,
				},
			};
		case "deploy/setExecutionTargets":
			return {
				...state,
				deploy: {
					...state.deploy,
					executionTargets: action.targets,
				},
			};
		case "deploy/setSelectedTargetId":
			return {
				...state,
				deploy: {
					...state.deploy,
					selectedTargetId: action.targetId,
				},
			};
		case "assetContext/setLoading":
			return {
				...state,
				assetContext: {
					...state.assetContext,
					selectedNodeAssetLoading: action.loading,
				},
			};
		case "assetContext/setAsset":
			return {
				...state,
				assetContext: {
					...state.assetContext,
					selectedNodeAsset: action.asset,
				},
			};
		case "assetContext/setError":
			return {
				...state,
				assetContext: {
					...state.assetContext,
					selectedNodeAssetError: action.error,
				},
			};
		case "assetContext/reset":
			return {
				...state,
				assetContext: {
					selectedNodeAsset: null,
					selectedNodeAssetLoading: false,
					selectedNodeAssetError: null,
				},
			};
		default:
			return state;
	}
}
