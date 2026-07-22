export interface Pipeline {
	name: string;
	version?: string;
	nodes: PipelineNodeDef[];
	edges: PipelineEdgeDef[];

	/** Asset input selection — which assets this pipeline processes. */
	assetSelection?: {
		assetIds?: string[];
		assetQuery?: string;
	};
}

export interface PipelineNodeDef {
	id: string;
	component: Component;
	inputs?: Port[];
	outputs?: Port[];
	runtimeConfig?: PipelineNodeRuntimeConfig;
	runtimeSecrets?: PipelineNodeRuntimeSecretMount[];
	storageMounts?: PipelineNodeRuntimeStorageMount[];
}

export interface PipelineNodeRuntimeConfig {
	mode: "saved";
	configId: string;
	version: number;
	fileName?: string;
	mountPath: string;
	targetFilename: string;
	displayName?: string;
}

export interface PipelineNodeRuntimeSecretMount {
	resourceId: string;
	mountPath?: string;
	displayName?: string;
}

export interface PipelineNodeRuntimeStorageMount {
	resourceId: string;
	mountPath?: string;
	readOnly?: boolean;
	displayName?: string;
}

export interface Component {
	name: string;
	image: string;
	type?: string;
	source?: string;
	command?: string[];
	args?: Argument[];
	env?: Argument[];
	resources?: ResourceRequirements;
	componentId?: string;
	releaseId?: string;
	componentVersionLabel?: string;
}

export interface Argument {
	name: string;
	value?: string;
	from?: string;
}

export interface PipelineEdgeDef {
	source: string;
	target: string;
}

export interface Port {
	name: string;
	type: string;
	desc?: string;
	default_value?: string;
}

export interface ResourceRequirements {
	cpu?: string;
	memory?: string;
	// Optional Burstable overrides: cpu/memory act as the request (or request==limit
	// when these are empty); *Limit sets a higher ceiling. request must be <= limit.
	cpuLimit?: string;
	memoryLimit?: string;
	disk?: string;
	gpu?: string;
	computeTier?: string;
	type?: string;
	source?: string;
	env?: Record<string, string>;
}

export interface RegisteredComponent {
	id: string;
	componentId?: string;
	releaseId?: string;
	name: string;
	type?: string;
	source?: string;
	image: string;
	tag?: string;
	releaseLabel?: string;
	sourceCommit?: string;
	imageUid?: string;
	command: string[];
	args: Argument[];
	env?: Argument[];
	inputPorts?: Port[];
	outputPorts?: Port[];
	cpu: string;
	memory: string;
	cpuLimit?: string;
	memoryLimit?: string;
	disk: string;
	gpu?: string;
	computeTier?: string;
}

export interface PipelineNodeData {
	label: string;
	type?: string;
	source?: string;
	image: string;
	command: string[];
	args: Argument[];
	env?: Argument[];
	inputPorts?: Port[];
	outputPorts?: Port[];
	cpu: string;
	memory: string;
	cpuLimit?: string;
	memoryLimit?: string;
	disk: string;
	gpu?: string;
	computeTier?: string;
	componentId?: string;
	releaseId?: string;
	componentVersionLabel?: string;
	runtimeConfig?: PipelineNodeRuntimeConfig;
	runtimeSecrets?: PipelineNodeRuntimeSecretMount[];
	storageMounts?: PipelineNodeRuntimeStorageMount[];
	[key: string]: unknown;
}
