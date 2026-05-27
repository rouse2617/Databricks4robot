import { request } from "./pipelineClient";

export interface PortDef {
  name: string;
  type: string;
  desc?: string;
  default_value?: string;
}

export interface EnvVarDef {
  name: string;
  value?: string;
}

/** Backend PipelineComponent model shape. */
export interface PipelineComponentAPI {
  id: string;
  name: string;
  description: string;
  image: string;
  tag: string;
  source: string;
  inputPorts: PortDef[];
  outputPorts: PortDef[];
  resources?: Record<string, unknown>;
  envVars?: EnvVarDef[];
  createdAt: string;
  updatedAt: string;
}

/** List all registered pipeline components. */
export function listComponents(): Promise<{ items: PipelineComponentAPI[] }> {
  return request("GET", "/components");
}

/** Create a new pipeline component. */
export function createComponent(
  pc: Omit<PipelineComponentAPI, "id" | "createdAt" | "updatedAt">,
): Promise<PipelineComponentAPI> {
  return request("POST", "/components", pc);
}

/** Update an existing pipeline component. */
export function updateComponent(
  id: string,
  pc: Partial<PipelineComponentAPI>,
): Promise<PipelineComponentAPI> {
  return request("PUT", `/components/${encodeURIComponent(id)}`, pc);
}

/** Delete a pipeline component. */
export function deleteComponent(id: string): Promise<void> {
  return request("DELETE", `/components/${encodeURIComponent(id)}`);
}
