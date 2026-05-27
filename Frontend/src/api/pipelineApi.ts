import { request } from "./pipelineClient";
import type { Pipeline } from "../components/pipeline/types";

export interface PipelineTemplate {
  id: string;
  name: string;
  pipeline: Pipeline;
  nodeCount: number;
  createdAt: string;
}

export interface Deployment {
  id: string;
  pipelineName: string;
  workflowName: string;
  status: string;
  nodeCount: number;
  createdAt: string;
  finishedAt?: string;
  manifest?: string;
  pipelineJSON?: Pipeline;
}

export function listPipelines(): Promise<PipelineTemplate[]> {
  return request<{ items: PipelineTemplate[] }>("GET", "/pipelines").then((r) => r.items);
}

export function getPipeline(id: string): Promise<PipelineTemplate> {
  return request<PipelineTemplate>("GET", `/pipelines/${id}`);
}

export function savePipeline(name: string, pipeline: unknown): Promise<PipelineTemplate> {
  return request<PipelineTemplate>("POST", "/pipelines", { name, pipeline });
}

export function deletePipeline(id: string): Promise<void> {
  return request<void>("DELETE", `/pipelines/${id}`);
}

export function deployTemplate(templateId: string, assetIds?: string[]): Promise<Deployment> {
  return request<Deployment>("POST", `/deploy/template/${templateId}`, { asset_ids: assetIds });
}

export function listDeployments(): Promise<Deployment[]> {
  return request<{ items: Deployment[] }>("GET", "/deployments").then((r) => r.items);
}

export function deleteDeployment(id: string): Promise<void> {
  return request<void>("DELETE", `/deployments/${id}`);
}
