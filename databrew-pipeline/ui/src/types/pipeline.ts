// Pipeline types — mirrors the Go transpiler model

export interface Pipeline {
  name: string;
  version?: string;
  nodes: PipelineNode[];
  edges: PipelineEdge[];
}

export interface PipelineNode {
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

export interface PipelineEdge {
  source: string;
  target: string;
}

export interface Port {
  name: string;
  type: string;
}

// PortDef extends Port with default value for component registry (F2.10).
export interface PortDef extends Port {
  default_value?: string;
}

export interface ResourceRequirements {
  cpu?: string;
  memory?: string;
  disk?: string;
}
