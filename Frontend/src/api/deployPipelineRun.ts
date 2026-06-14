import { type BatchJob, createBatchJob } from "./batchJobApi";
import {
	batchAssetLimitError,
	exceedsBatchAssetLimit,
} from "../lib/batchAssetLimits";
import {
	type Deployment,
	deployTemplate,
	normalizeDeployResults,
} from "./pipelineApi";

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
	},
): Promise<DeployPipelineResult> {
	if (exceedsBatchAssetLimit(assetIds.length)) {
		throw new Error(batchAssetLimitError(assetIds.length));
	}
	if (assetIds.length >= BATCH_ASSET_THRESHOLD) {
		const stamp = new Date()
			.toISOString()
			.slice(0, 19)
			.replace(/[-:T]/g, "");
		const batchJob = await createBatchJob({
			name: options?.batchName ?? `batch-${stamp}`,
			templateId,
			assetIds,
		});
		return { mode: "batch", batchJob };
	}

	const result = await deployTemplate(
		templateId,
		assetIds,
		options?.targetId,
		options?.version,
	);
	return { mode: "single", runs: normalizeDeployResults(result) };
}
