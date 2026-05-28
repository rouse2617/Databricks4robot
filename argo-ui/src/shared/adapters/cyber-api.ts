import * as superagent from 'superagent';

import requests from '../services/requests';

const CYBER_BASE = 'api/v1';

export interface PipelineTemplate {
    id: string;
    name: string;
    version: number;
    pipeline: Record<string, unknown>;
    nodeCount: number;
    createdAt: string;
    updatedAt: string;
}

export interface PipelineDeployment {
    id: string;
    templateId?: string;
    pipelineName: string;
    workflowName: string;
    status: string;
    nodeCount: number;
    manifest?: string;
    pipelineJSON?: Record<string, unknown>;
    createdAt: string;
    updatedAt: string;
    finishedAt?: string;
}

export interface CyberWorkflow {
    metadata: {
        name: string;
        namespace: string;
        uid?: string;
        creationTimestamp?: string;
        labels?: Record<string, string>;
    };
    status: {
        phase: string;
        message?: string;
        startedAt?: string;
        finishedAt?: string;
    };
}

export interface CyberWorkflowList {
    items: CyberWorkflow[];
}

/** Cyber pipeline & workflow API adapter.
 *
 * Wraps Cyber backend endpoints and maps between Argo and Cyber models.
 * Fallback to polling when SSE is unavailable.
 */
export const CyberApi = {
    /** Deploy a pipeline (equivalent to Argo submit). */
    async deploy(pipeline: Record<string, unknown>, name?: string, assetIDs?: string[]): Promise<PipelineDeployment> {
        const res = await requests
            .post(`${CYBER_BASE}/deploy`)
            .send({pipeline, name, asset_ids: assetIDs});
        return res.body as PipelineDeployment;
    },

    /** Deploy a saved pipeline template by ID. */
    async deployByTemplate(templateId: string): Promise<PipelineDeployment> {
        const res = await requests
            .post(`${CYBER_BASE}/deploy/template/${encodeURIComponent(templateId)}`);
        return res.body as PipelineDeployment;
    },

    /** List pipeline templates. */
    async listTemplates(): Promise<{items: PipelineTemplate[]}> {
        const res = await requests.get(`${CYBER_BASE}/pipelines`);
        return res.body as {items: PipelineTemplate[]};
    },

    /** Get a single pipeline template. */
    async getTemplate(id: string): Promise<PipelineTemplate> {
        const res = await requests.get(`${CYBER_BASE}/pipelines/${encodeURIComponent(id)}`);
        return res.body as PipelineTemplate;
    },

    /** List deployments (runs). */
    async listDeployments(): Promise<{items: PipelineDeployment[]}> {
        const res = await requests.get(`${CYBER_BASE}/deployments`);
        return res.body as {items: PipelineDeployment[]};
    },

    /** Get a single deployment. */
    async getDeployment(id: string): Promise<PipelineDeployment> {
        const res = await requests.get(`${CYBER_BASE}/deployments/${encodeURIComponent(id)}`);
        return res.body as PipelineDeployment;
    },

    /** List workflows (monitoring). */
    async listWorkflows(): Promise<CyberWorkflowList> {
        const res = await requests.get(`${CYBER_BASE}/workflows`);
        return res.body as CyberWorkflowList;
    },

    /** Get a single workflow by name. */
    async getWorkflow(name: string): Promise<CyberWorkflow> {
        const res = await requests.get(`${CYBER_BASE}/workflows/${encodeURIComponent(name)}`);
        return res.body as CyberWorkflow;
    },

    /** Retry a workflow. */
    async retryWorkflow(name: string): Promise<void> {
        await requests.post(`${CYBER_BASE}/workflows/${encodeURIComponent(name)}/retry`);
    },

    /** Suspend a workflow. */
    async suspendWorkflow(name: string): Promise<void> {
        await requests.post(`${CYBER_BASE}/workflows/${encodeURIComponent(name)}/suspend`);
    },

    /** Resume a workflow. */
    async resumeWorkflow(name: string): Promise<void> {
        await requests.post(`${CYBER_BASE}/workflows/${encodeURIComponent(name)}/resume`);
    },

    /** Terminate a workflow. */
    async terminateWorkflow(name: string): Promise<void> {
        await requests.post(`${CYBER_BASE}/workflows/${encodeURIComponent(name)}/terminate`);
    },

    /** Delete a workflow. */
    async deleteWorkflow(name: string): Promise<void> {
        await requests.delete(`${CYBER_BASE}/workflows/${encodeURIComponent(name)}`);
    }
};
