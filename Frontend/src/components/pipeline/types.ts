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
}

export interface Component {
	name: string;
	image: string;
	type?: string;
	source?: string;
	command?: string[];
	args?: Argument[];
	env?: Record<string, string>;
	resources?: ResourceRequirements;
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
	disk?: string;
	gpu?: string;
	computeTier?: string;
	type?: string;
	source?: string;
	env?: Record<string, string>;
}

export interface RegisteredComponent {
	id: string;
	name: string;
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
	disk: string;
	gpu?: string;
	computeTier?: string;
	[key: string]: unknown;
}
