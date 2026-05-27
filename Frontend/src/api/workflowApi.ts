const API = "/api/v1";

export interface WorkflowSummary {
  name: string;
  status: string;
  nodeCount: number;
  createdAt: string;
  finishedAt?: string;
}

export interface WorkflowNodeStatus {
  id: string;
  name: string;
  displayName: string;
  phase: string;
  message?: string;
  startedAt?: string;
  finishedAt?: string;
}

export interface WorkflowDetail {
  name: string;
  status: string;
  message?: string;
  nodes: WorkflowNodeStatus[];
  createdAt: string;
  finishedAt?: string;
}

class ApiError extends Error {
  code: string;
  status: number;
  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

async function request<T>(method: string, path: string): Promise<T> {
  const res = await fetch(`${API}${path}`, { method });
  if (!res.ok) {
    let code = "UNKNOWN";
    let message = `HTTP ${res.status}`;
    try {
      const err = await res.json();
      code = err.code || code;
      message = err.message || err.error || message;
    } catch {
      /* ignore parse errors */
    }
    throw new ApiError(res.status, code, message);
  }
  return res.json() as Promise<T>;
}

export function listWorkflows(): Promise<{ items: WorkflowSummary[] }> {
  return request("GET", "/workflows");
}

export function getWorkflow(name: string): Promise<WorkflowDetail> {
  return request("GET", `/workflows/${encodeURIComponent(name)}`);
}

export function getWorkflowLogs(
  name: string,
  nodeId: string,
): Promise<{ logs: string }> {
  return request(
    "GET",
    `/workflows/${encodeURIComponent(name)}/logs?nodeId=${encodeURIComponent(nodeId)}`,
  );
}
