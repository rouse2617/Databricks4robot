package argo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// WorkflowClient defines the interface for managing Argo Workflows.
type WorkflowClient interface {
	CreateWorkflow(ctx context.Context, wf *wfv1.Workflow, namespace string) error
	GetWorkflowStatus(ctx context.Context, name, namespace string) (wfv1.WorkflowPhase, error)
	DeleteWorkflow(ctx context.Context, name, namespace string) error
	ListWorkflows(ctx context.Context, namespace string, labelSelector string) ([]wfv1.Workflow, error)
	GetWorkflow(ctx context.Context, name, namespace string) (*wfv1.Workflow, error)
	StopWorkflow(ctx context.Context, name, namespace string) error
	RetryWorkflow(ctx context.Context, name, namespace string) error
	ResubmitWorkflow(ctx context.Context, name, namespace string) error
	SuspendWorkflow(ctx context.Context, name, namespace string) error
	ResumeWorkflow(ctx context.Context, name, namespace string) error
	TerminateWorkflow(ctx context.Context, name, namespace string) error
	GetWorkflowLogs(ctx context.Context, workflowName, nodeId, namespace string) (string, error)
}

// Client implements WorkflowClient using the Argo Server REST API.
type Client struct {
	serverURL  string
	token      string
	httpClient *http.Client
}

// CreateWorkflow creates a workflow in the specified namespace.
// Sends a WorkflowCreateRequest body ({workflow: ...}) per Argo Server v4 REST API.
func (c *Client) CreateWorkflow(ctx context.Context, wf *wfv1.Workflow, namespace string) error {
	if wf == nil {
		return fmt.Errorf("workflow is nil")
	}
	return c.do(ctx, http.MethodPost, workflowPath(namespace), nil,
		map[string]any{"workflow": wf}, nil)
}

// GetWorkflowStatus returns the current phase of the named workflow.
func (c *Client) GetWorkflowStatus(ctx context.Context, name, namespace string) (wfv1.WorkflowPhase, error) {
	wf, err := c.GetWorkflow(ctx, name, namespace)
	if err != nil {
		return wfv1.WorkflowUnknown, err
	}
	return wf.Status.Phase, nil
}

// DeleteWorkflow removes a workflow by name.
func (c *Client) DeleteWorkflow(ctx context.Context, name, namespace string) error {
	return c.do(ctx, http.MethodDelete, workflowNamePath(namespace, name), nil, nil, nil)
}

// ListWorkflows returns workflows in the namespace filtered by label selector.
func (c *Client) ListWorkflows(ctx context.Context, namespace string, labelSelector string) ([]wfv1.Workflow, error) {
	query := url.Values{}
	if labelSelector != "" {
		query.Set("listOptions.labelSelector", labelSelector)
	}

	var list wfv1.WorkflowList
	if err := c.do(ctx, http.MethodGet, workflowPath(namespace), query, nil, &list); err != nil {
		return nil, err
	}
	return list.Items, nil
}

// GetWorkflow returns the full workflow object by name.
func (c *Client) GetWorkflow(ctx context.Context, name, namespace string) (*wfv1.Workflow, error) {
	var wf wfv1.Workflow
	if err := c.do(ctx, http.MethodGet, workflowNamePath(namespace, name), nil, nil, &wf); err != nil {
		return nil, err
	}
	return &wf, nil
}

// StopWorkflow stops a running workflow through Argo Server's stop endpoint.
func (c *Client) StopWorkflow(ctx context.Context, name, namespace string) error {
	return c.do(ctx, http.MethodPut, workflowNamePath(namespace, name)+"/stop", nil, map[string]any{}, nil)
}

// RetryWorkflow retries a workflow through Argo Server's retry endpoint.
func (c *Client) RetryWorkflow(ctx context.Context, name, namespace string) error {
	return c.workflowOperation(ctx, name, namespace, "retry")
}

// ResubmitWorkflow resubmits a workflow through Argo Server's resubmit endpoint.
func (c *Client) ResubmitWorkflow(ctx context.Context, name, namespace string) error {
	return c.workflowOperation(ctx, name, namespace, "resubmit")
}

// SuspendWorkflow suspends a workflow through Argo Server's suspend endpoint.
func (c *Client) SuspendWorkflow(ctx context.Context, name, namespace string) error {
	return c.workflowOperation(ctx, name, namespace, "suspend")
}

// ResumeWorkflow resumes a workflow through Argo Server's resume endpoint.
func (c *Client) ResumeWorkflow(ctx context.Context, name, namespace string) error {
	return c.workflowOperation(ctx, name, namespace, "resume")
}

// TerminateWorkflow terminates a workflow through Argo Server's terminate endpoint.
func (c *Client) TerminateWorkflow(ctx context.Context, name, namespace string) error {
	return c.workflowOperation(ctx, name, namespace, "terminate")
}

func (c *Client) workflowOperation(ctx context.Context, name, namespace, operation string) error {
	return c.do(ctx, http.MethodPut, workflowNamePath(namespace, name)+"/"+operation, nil, map[string]any{}, nil)
}

// GetWorkflowLogs returns logs for a workflow node using Argo Server log streaming.
func (c *Client) GetWorkflowLogs(ctx context.Context, workflowName, nodeId, namespace string) (string, error) {
	query := url.Values{}
	query.Set("logOptions.container", "main")
	if nodeId != "" {
		query.Set("grep", nodeId)
	}

	resp, err := c.doRequest(ctx, http.MethodGet, workflowNamePath(namespace, workflowName)+"/log", query, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	logs, err := parseLogStream(resp.Body)
	if err != nil {
		return "", fmt.Errorf("parse workflow logs: %w", err)
	}
	return logs, nil
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	resp, err := c.doRequest(ctx, method, path, query, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if out == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode argo response: %w", err)
	}
	return nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, query url.Values, body any) (*http.Response, error) {
	if c == nil {
		return nil, fmt.Errorf("argo client is nil")
	}
	if c.serverURL == "" {
		return nil, fmt.Errorf("argo server URL is empty")
	}

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode argo request: %w", err)
		}
		reader = bytes.NewReader(raw)
	}

	endpoint, err := c.url(path, query)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, fmt.Errorf("create argo request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", authorizationHeader(c.token))
	}

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call argo API: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		defer resp.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		message := strings.TrimSpace(string(raw))
		if message == "" {
			message = resp.Status
		}
		return nil, fmt.Errorf("argo API %s %s failed: %s", method, path, message)
	}
	return resp, nil
}

func (c *Client) url(path string, query url.Values) (string, error) {
	base, err := url.Parse(c.serverURL)
	if err != nil {
		return "", fmt.Errorf("parse argo server URL: %w", err)
	}
	pathURL, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("parse argo API path: %w", err)
	}
	resolved := base.ResolveReference(pathURL)
	if len(query) > 0 {
		resolved.RawQuery = query.Encode()
	}
	return resolved.String(), nil
}

func workflowPath(namespace string) string {
	return "/api/v1/workflows/" + url.PathEscape(namespace)
}

func workflowNamePath(namespace, name string) string {
	return workflowPath(namespace) + "/" + url.PathEscape(name)
}

func authorizationHeader(token string) string {
	if strings.Contains(token, " ") {
		return token
	}
	return "Bearer " + token
}
