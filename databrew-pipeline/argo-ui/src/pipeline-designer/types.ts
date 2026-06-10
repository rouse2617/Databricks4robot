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
  imagePullPolicy?: string;
  command?: string[];
  args?: Argument[];
  env?: EnvVar[];
  resources?: ResourceRequirements;
}

export interface Argument {
  name: string;
  value?: string;
  from?: string;
}

export interface EnvVar {
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

export interface ResourceRequirements {
  cpu?: string;
  memory?: string;
  disk?: string;
}
