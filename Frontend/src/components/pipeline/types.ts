export interface Pipeline {
	name: string;
	version?: string;
	nodes: PipelineNodeDef[];
	edges: PipelineEdgeDef[];
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
	command?: string[];
	args?: Argument[];
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
}

export interface ResourceRequirements {
	cpu?: string;
	memory?: string;
	disk?: string;
}

export interface RegisteredComponent {
	id: string;
	name: string;
	image: string;
	command: string[];
	args: Argument[];
	cpu: string;
	memory: string;
	disk: string;
}

export interface PipelineNodeData {
	label: string;
	image: string;
	command: string[];
	args: Argument[];
	cpu: string;
	memory: string;
	disk: string;
	[key: string]: unknown;
}
