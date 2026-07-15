package argo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// ErrNotFound indicates the requested Argo resource does not exist.
var ErrNotFound = errors.New("argo resource not found")

// ErrUnexpectedNotFound indicates a 404 response that did NOT come from the
// Argo API server — likely a misconfigured base URL pointing to a non-Argo
// service (e.g. the DataBrew backend or pipeline UI). Callers MUST NOT treat
// this as "workflow genuinely not found" and MUST NOT write terminal status.
var ErrUnexpectedNotFound = errors.New("argo unexpected not found: response is not from Argo API")

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
	GetWorkflowLogs(ctx context.Context, workflowName, podName, namespace string, opts WorkflowLogOptions) (WorkflowLogResult, error)
	GetWorkflowLogStream(ctx context.Context, workflowName, podName, namespace string, opts WorkflowLogOptions) (io.ReadCloser, error)
}

// WorkflowResubmitResultClient exposes Argo's resubmit response, which contains
// the newly created workflow object.
type WorkflowResubmitResultClient interface {
	ResubmitWorkflowWithResult(ctx context.Context, name, namespace string) (*wfv1.Workflow, error)
}

// WorkflowLogOptions contains bounded pod log query options.
type WorkflowLogOptions struct {
	Container    string
	TailLines    *int64
	LimitBytes   *int64
	SinceSeconds *int64
	SinceTime    string
	Previous     bool
	Timestamps   bool
	Follow       bool
}

// WorkflowLogResult contains parsed log text plus truncation metadata.
type WorkflowLogResult struct {
	Logs       string
	LineCount  int
	Truncated  bool
	LimitBytes int64
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
	_, err := c.ResubmitWorkflowWithResult(ctx, name, namespace)
	return err
}

// ResubmitWorkflowWithResult resubmits a workflow and returns the new workflow
// object created by Argo.
func (c *Client) ResubmitWorkflowWithResult(ctx context.Context, name, namespace string) (*wfv1.Workflow, error) {
	var wf wfv1.Workflow
	if err := c.do(ctx, http.MethodPut, workflowNamePath(namespace, name)+"/resubmit", nil, map[string]any{}, &wf); err != nil {
		return nil, err
	}
	return &wf, nil
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

// GetWorkflowLogs returns bounded logs for a workflow pod using Argo Server log streaming.
func (c *Client) GetWorkflowLogs(
	ctx context.Context,
	workflowName, podName, namespace string,
	opts WorkflowLogOptions,
) (WorkflowLogResult, error) {
	query := workflowLogQuery(podName, opts)

	resp, err := c.doRequest(ctx, http.MethodGet, workflowNamePath(namespace, workflowName)+"/log", query, nil)
	if err != nil {
		return WorkflowLogResult{}, err
	}
	defer resp.Body.Close()

	logs, lineCount, truncated, err := parseLogStreamBounded(resp.Body, opts.LimitBytes)
	if err != nil {
		return WorkflowLogResult{}, fmt.Errorf("parse workflow logs: %w", err)
	}
	result := WorkflowLogResult{
		Logs:      logs,
		LineCount: lineCount,
		Truncated: truncated,
	}
	if opts.LimitBytes != nil {
		result.LimitBytes = *opts.LimitBytes
	}
	return result, nil
}

// GetWorkflowLogStream returns a live log stream for a specific workflow pod.
func (c *Client) GetWorkflowLogStream(
	ctx context.Context,
	workflowName, podName, namespace string,
	opts WorkflowLogOptions,
) (io.ReadCloser, error) {
	// Follow is decided by the caller (StreamWorkflowLogs only follows nodes that
	// are still running). Forcing it true here made log requests for finished
	// nodes hang until the Cloud Run request timeout (Argo's follow never EOFs a
	// completed workflow), returning 504 and starving the browser connection pool
	// via client auto-reconnect. (CYB-3483)
	query := workflowLogQuery(podName, opts)

	resp, err := c.doRequest(ctx, http.MethodGet, workflowNamePath(namespace, workflowName)+"/log", query, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func workflowLogQuery(podName string, opts WorkflowLogOptions) url.Values {
	query := url.Values{}
	query.Set("podName", podName)
	if opts.Container != "" {
		query.Set("container", opts.Container)
		query.Set("logOptions.container", opts.Container)
	}
	if opts.TailLines != nil {
		query.Set("logOptions.tailLines", fmt.Sprintf("%d", *opts.TailLines))
	}
	if opts.LimitBytes != nil {
		query.Set("logOptions.limitBytes", fmt.Sprintf("%d", *opts.LimitBytes))
	}
	if opts.SinceSeconds != nil {
		query.Set("logOptions.sinceSeconds", fmt.Sprintf("%d", *opts.SinceSeconds))
	}
	if opts.SinceTime != "" {
		query.Set("logOptions.sinceTime", opts.SinceTime)
	}
	if opts.Previous {
		query.Set("logOptions.previous", "true")
	}
	if opts.Timestamps {
		query.Set("logOptions.timestamps", "true")
	}
	if opts.Follow {
		query.Set("follow", "true")
		query.Set("logOptions.follow", "true")
	}
	return query
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
		if resp.StatusCode == http.StatusNotFound {
			// Distinguish between a genuine Argo API 404 (JSON with gRPC
			// code=5) and a non-Argo 404 (HTML, plain text, or JSON with
			// code=404) that indicates a misconfigured base URL.
			if IsArgo404Response(message) {
				return nil, fmt.Errorf("%w: %s", ErrNotFound, message)
			}
			return nil, fmt.Errorf("%w: body=%q", ErrUnexpectedNotFound, message)
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

// IsArgo404Response returns true when a 404 response body originates from
// the Argo API server (JSON with gRPC code 5) versus a non-Argo service
// such as the DataBrew backend (HTML, plain text, or JSON with code 404).
func IsArgo404Response(body string) bool {
	return strings.Contains(body, `"code":5`) || strings.Contains(body, `"code": 5`)
}
