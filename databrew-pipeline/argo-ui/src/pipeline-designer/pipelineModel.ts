import type { Edge, Node } from '@xyflow/react';
import type { Argument, EnvVar, Pipeline, PipelineEdge, PipelineNode, Port, ResourceRequirements } from './types';

export const DEFAULT_INPUT_PORT = 'in';
export const DEFAULT_OUTPUT_PORT = 'out';

export function defaultInputs(): Port[] {
  return [{ name: DEFAULT_INPUT_PORT, type: 'string' }];
}

export function defaultOutputs(): Port[] {
  return [{ name: DEFAULT_OUTPUT_PORT, type: 'string' }];
}

export function splitRef(ref: string): [string, string] {
  const dot = ref.lastIndexOf('.');
  if (dot < 0) return [ref, ''];
  return [ref.slice(0, dot), ref.slice(dot + 1)];
}

export function edgeRef(nodeId: string, port: string): string {
  return `${nodeId}.${port}`;
}

export function buildResources(cpu?: string, memory?: string, disk?: string): ResourceRequirements | undefined {
  const res: ResourceRequirements = {};
  if (cpu?.trim()) res.cpu = cpu.trim();
  if (memory?.trim()) res.memory = memory.trim();
  if (disk?.trim()) res.disk = disk.trim();
  return Object.keys(res).length > 0 ? res : undefined;
}

export function nodeDataToPipelineNode(n: Node): PipelineNode {
  const d = n.data as Record<string, unknown>;
  const inputs = (d.inputs as Port[] | undefined)?.length ? (d.inputs as Port[]) : defaultInputs();
  const outputs = (d.outputs as Port[] | undefined)?.length ? (d.outputs as Port[]) : defaultOutputs();
  return {
    id: n.id,
    component: {
      name: (d.label as string) || '',
      image: (d.image as string) || '',
      imagePullPolicy: (d.imagePullPolicy as string) || undefined,
      command: (d.command as string[]) || [],
      args: (d.args as Argument[]) || [],
      env: (d.env as EnvVar[]) || [],
      resources: buildResources(d.cpu as string, d.memory as string, d.disk as string),
    },
    inputs,
    outputs,
  };
}

export function edgeToPipelineEdge(e: Edge): PipelineEdge {
  const srcPort = e.sourceHandle || DEFAULT_OUTPUT_PORT;
  const tgtPort = e.targetHandle || DEFAULT_INPUT_PORT;
  return {
    source: edgeRef(e.source, srcPort),
    target: edgeRef(e.target, tgtPort),
  };
}

export function buildPipeline(name: string, nodes: Node[], edges: Edge[]): Pipeline {
  return {
    name,
    version: '1',
    nodes: nodes.map(nodeDataToPipelineNode),
    edges: edges.map(edgeToPipelineEdge),
  };
}

export function pipelineNodeToFlowNode(pn: PipelineNode, index: number): Node {
  return {
    id: pn.id,
    type: 'pipelineStep',
    position: { x: 120 + (index % 4) * 220, y: 80 + Math.floor(index / 4) * 140 },
    data: {
      label: pn.component.name,
      image: pn.component.image,
      imagePullPolicy: pn.component.imagePullPolicy || '',
      command: pn.component.command || [],
      args: pn.component.args || [],
      env: pn.component.env || [],
      cpu: pn.component.resources?.cpu || '',
      memory: pn.component.resources?.memory || '',
      disk: pn.component.resources?.disk || '',
      inputs: pn.inputs?.length ? pn.inputs : defaultInputs(),
      outputs: pn.outputs?.length ? pn.outputs : defaultOutputs(),
    },
  };
}

export function pipelineEdgeToFlowEdge(pe: PipelineEdge, index: number): Edge {
  const [srcNode, srcPort] = splitRef(pe.source);
  const [tgtNode, tgtPort] = splitRef(pe.target);
  return {
    id: `e-${index}`,
    source: srcNode,
    target: tgtNode,
    sourceHandle: srcPort || DEFAULT_OUTPUT_PORT,
    targetHandle: tgtPort || DEFAULT_INPUT_PORT,
  };
}

export function dropOffset(count: number): { dx: number; dy: number } {
  return { dx: (count % 4) * 48, dy: Math.floor(count / 4) * 48 };
}
