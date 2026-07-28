import {
	batchAssetLimitError,
	exceedsBatchAssetLimit,
} from "../lib/batchAssetLimits";
import { type BatchJob, createBatchJob } from "./batchJobApi";
import {
	type DeployConfigSelection,
	type Deployment,
	normalizeDeployResults,
} from "./pipelineApi";
import { createRunByTemplate } from "./runApi";

export const BATCH_ASSET_THRESHOLD = 2;

export type DeployPipelineResult =
	| { mode: "single"; runs: Deployment[] }
	| { mode: "batch"; batchJob: BatchJob };

export async function deployPipelineForAssets(
	templateId: string,
	assetIds: string[],
	options?: {
		targetId?: string;
		version?: number;
		batchName?: string;
		configSelection?: DeployConfigSelection;
		// Per-dispatch Argo priority override (batch path only; a single run
		// inherits the target pool default). Omit to inherit the pool default.
		priority?: number;
	},
): Promise<DeployPipelineResult> {
	if (exceedsBatchAssetLimit(assetIds.length)) {
		throw new Error(batchAssetLimitError(assetIds.length));
	}
	if (assetIds.length >= BATCH_ASSET_THRESHOLD) {
		const stamp = new Date().toISOString().slice(0, 19).replace(/[-:T]/g, "");
		const batchJob = await createBatchJob({
			name: options?.batchName ?? `batch-${stamp}`,
			templateId,
			assetIds,
			targetId: options?.targetId,
			templateVersion: options?.version,
			configSelection: options?.configSelection,
			priority: options?.priority,
		});
		return { mode: "batch", batchJob };
	}

	const result = await createRunByTemplate(
		templateId,
		assetIds,
		options?.targetId,
		options?.version,
		options?.configSelection,
	);
	return { mode: "single", runs: normalizeDeployResults(result) };
}
