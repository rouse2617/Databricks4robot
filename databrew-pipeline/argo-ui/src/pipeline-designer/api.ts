const API = '/api';

export interface PipelineTemplate {
  id: string;
  name: string;
  pipeline: unknown;
  nodeCount: number;
  createdAt: string;
}

export interface Deployment {
  id: string;
  pipelineName: string;
  workflowName: string;
  status: string;
  nodes: number;
  createdAt: string;
  finishedAt?: string;
  manifest?: string;
  pipelineJSON?: string;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${API}${path}`, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  const json = await res.json();
  if (!json.ok) throw new Error(json.error || 'request failed');
  return json.data as T;
}

export function listPipelines(): Promise<PipelineTemplate[]> {
  return request('GET', '/pipelines');
}

export function getPipeline(id: string): Promise<PipelineTemplate> {
  return request('GET', `/pipeline/${id}`);
}

export function savePipeline(name: string, pipeline: unknown): Promise<PipelineTemplate> {
  return request('POST', '/pipelines', { name, pipeline });
}

export function deletePipeline(id: string): Promise<void> {
  return request('DELETE', `/pipelines/${id}`);
}

export function deploy(pipeline: unknown, name?: string): Promise<Deployment> {
  return request('POST', '/deploy', { pipeline, name });
}

export function deployTemplate(templateId: string): Promise<Deployment> {
  return request('POST', '/deploy', { templateId });
}

export function listDeployments(): Promise<Deployment[]> {
  return request('GET', '/deployments');
}

export function getDeployment(id: string): Promise<Deployment> {
  return request('GET', `/deployments/${id}`);
}

export function deleteDeployment(id: string): Promise<void> {
  return request('DELETE', `/deployments/${id}`);
}
