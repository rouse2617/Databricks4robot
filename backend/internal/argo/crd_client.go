package argo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// workflowGVR is the argoproj.io Workflow CRD GroupVersionResource. Argo v3.x
// stays on v1alpha1; a future v4 bump would need re-mapping (Non-goal per
// design doc D3).
var workflowGVR = schema.GroupVersionResource{
	Group:    "argoproj.io",
	Version:  "v1alpha1",
	Resource: "workflows",
}

// errCRDMethodNotImplemented is returned by crdWorkflowClient methods that
// belong to later PRs in the 4d series (lifecycle ops → 4d.2, retry/resubmit
// → 4d.3, log streaming → 4d.4). Kept as a sentinel so tests can assert the
// scope boundary explicitly.
var errCRDMethodNotImplemented = errors.New("argo crd client: method not implemented in PR 4d.1 skeleton")

// crdWorkflowClient talks to argo-workflows directly via the K8s Workflow CRD
// (argoproj.io/v1alpha1) using dynamic + typed clients handed in by
// k8s.ClientFactory. It exists so DataBrew can reach clusters that do not
// expose argo-server on a network the backend can dial — see design doc D1/D2
// in openspec/changes/CYB-3486d-argo-crd-mode/design.md.
type crdWorkflowClient struct {
	dyn       dynamic.Interface
	pods      kubernetes.Interface // reserved for 4d.4 log streaming; unused today
	defaultNS string               // fallback for List calls without explicit ns
}

// newCRDWorkflowClient constructs the CRD-mode client. Callers get it from
// argo.ClientFactory when the cluster row lacks a usable argo-server URL.
func newCRDWorkflowClient(dyn dynamic.Interface, pods kubernetes.Interface, defaultNS string) *crdWorkflowClient {
	return &crdWorkflowClient{dyn: dyn, pods: pods, defaultNS: defaultNS}
}

var _ WorkflowClient = (*crdWorkflowClient)(nil)
var _ WorkflowResubmitResultClient = (*crdWorkflowClient)(nil)

// CreateWorkflow posts a new Workflow CRD to the cluster. The workflow's
// TypeMeta is normalized to argoproj.io/v1alpha1 Workflow so callers can hand
// in transpiler output without setting those fields themselves.
func (c *crdWorkflowClient) CreateWorkflow(ctx context.Context, wf *wfv1.Workflow, namespace string) error {
	if wf == nil {
		return fmt.Errorf("workflow is nil")
	}
	ns := c.resolveNS(namespace)
	u, err := workflowToUnstructured(wf)
	if err != nil {
		return err
	}
	if _, err := c.dyn.Resource(workflowGVR).Namespace(ns).Create(ctx, u, metav1.CreateOptions{}); err != nil {
		return translateK8sErr(err)
	}
	return nil
}

// GetWorkflow reads a Workflow CRD by name and decodes it into wfv1.Workflow
// so the rest of the backend sees the same type argo-server would return.
func (c *crdWorkflowClient) GetWorkflow(ctx context.Context, name, namespace string) (*wfv1.Workflow, error) {
	ns := c.resolveNS(namespace)
	u, err := c.dyn.Resource(workflowGVR).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, translateK8sErr(err)
	}
	return unstructuredToWorkflow(u)
}

// GetWorkflowStatus is a convenience over GetWorkflow that extracts the phase.
func (c *crdWorkflowClient) GetWorkflowStatus(ctx context.Context, name, namespace string) (wfv1.WorkflowPhase, error) {
	wf, err := c.GetWorkflow(ctx, name, namespace)
	if err != nil {
		return wfv1.WorkflowUnknown, err
	}
	return wf.Status.Phase, nil
}

// DeleteWorkflow removes a Workflow CRD by name.
func (c *crdWorkflowClient) DeleteWorkflow(ctx context.Context, name, namespace string) error {
	ns := c.resolveNS(namespace)
	if err := c.dyn.Resource(workflowGVR).Namespace(ns).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return translateK8sErr(err)
	}
	return nil
}

// ListWorkflows lists Workflow CRDs in the namespace, filtered by label
// selector. Empty selector returns everything the caller can see.
func (c *crdWorkflowClient) ListWorkflows(ctx context.Context, namespace, labelSelector string) ([]wfv1.Workflow, error) {
	ns := c.resolveNS(namespace)
	list, err := c.dyn.Resource(workflowGVR).Namespace(ns).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, translateK8sErr(err)
	}
	out := make([]wfv1.Workflow, 0, len(list.Items))
	for i := range list.Items {
		wf, err := unstructuredToWorkflow(&list.Items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, *wf)
	}
	return out, nil
}

// StopWorkflow patches spec.shutdown="Stop". workflow-controller sees the
// spec change on next reconcile and starts graceful shutdown: already-running
// pods continue until natural completion, no new pods are created, the
// workflow enters Stopped phase once all in-flight nodes finish. Aligned with
// argo-server /stop endpoint semantics.
func (c *crdWorkflowClient) StopWorkflow(ctx context.Context, name, namespace string) error {
	return c.patchSpec(ctx, name, namespace, []byte(`{"spec":{"shutdown":"Stop"}}`))
}

// TerminateWorkflow patches spec.shutdown="Terminate". workflow-controller
// immediately deletes all running pods and marks the workflow Failed with
// TerminationRequested. Aligned with argo-server /terminate.
func (c *crdWorkflowClient) TerminateWorkflow(ctx context.Context, name, namespace string) error {
	return c.patchSpec(ctx, name, namespace, []byte(`{"spec":{"shutdown":"Terminate"}}`))
}

// SuspendWorkflow patches spec.suspend=true. workflow-controller holds off
// creating new pods for pending nodes; already-running pods finish normally.
// Workflow phase remains Running — the suspend flag itself is what gates
// new-node dispatch. Aligned with argo-server /suspend.
func (c *crdWorkflowClient) SuspendWorkflow(ctx context.Context, name, namespace string) error {
	return c.patchSpec(ctx, name, namespace, []byte(`{"spec":{"suspend":true}}`))
}

// ResumeWorkflow patches spec.suspend=false. workflow-controller resumes
// dispatching queued nodes. Aligned with argo-server /resume.
func (c *crdWorkflowClient) ResumeWorkflow(ctx context.Context, name, namespace string) error {
	return c.patchSpec(ctx, name, namespace, []byte(`{"spec":{"suspend":false}}`))
}

// patchSpec runs a JSON-merge patch against the Workflow CRD. Merge patch is
// enough here because all lifecycle ops touch scalar fields under spec; the
// alternative (strategic merge) would require the argo scheme registered on
// the API server side, which is already the case, but merge patch is simpler
// and the payload is a static literal so nothing surprising interpolates in.
func (c *crdWorkflowClient) patchSpec(ctx context.Context, name, namespace string, patch []byte) error {
	ns := c.resolveNS(namespace)
	_, err := c.dyn.Resource(workflowGVR).Namespace(ns).
		Patch(ctx, name, types.MergePatchType, patch, metav1.PatchOptions{})
	return translateK8sErr(err)
}

// RetryWorkflow re-runs the failed nodes of a completed workflow. This is a
// hand-rolled subset of argo-server /retry (design D3 "simplified scope"): we
// don't pull in `argoproj/argo-workflows/v3/workflow/util` because that file
// transitively imports HDFS / Kerberos / OpenTelemetry — huge blast radius
// for a helper we barely need. Semantics we do provide:
//   - workflow must be in Failed / Error / Succeeded phase (i.e. completed)
//   - failed & errored nodes reset to empty phase so controller reruns them
//   - their pods are deleted (workflow-controller would otherwise skip retry
//     when it sees a stale Failed pod)
//   - workflow status phase / finishedAt / message reset so controller
//     re-picks it up
//
// Semantics NOT provided (deferred, do via argo CLI if needed): restart of
// successful nodes, partial retry via nodeFieldSelector, parameter overrides.
func (c *crdWorkflowClient) RetryWorkflow(ctx context.Context, name, namespace string) error {
	ns := c.resolveNS(namespace)
	wf, err := c.GetWorkflow(ctx, name, ns)
	if err != nil {
		return err
	}
	if !isRetryable(wf.Status.Phase) {
		return fmt.Errorf("argo crd: workflow %q not retryable (phase=%s)", name, wf.Status.Phase)
	}

	// Reset workflow-level status so the controller starts a fresh reconcile.
	wf.Status.Phase = wfv1.WorkflowRunning
	wf.Status.Message = ""
	wf.Status.FinishedAt = metav1.Time{}

	// Reset failed / errored nodes, collect pods to delete.
	var podsToDelete []string
	for id, node := range wf.Status.Nodes {
		if node.Phase != wfv1.NodeFailed && node.Phase != wfv1.NodeError {
			continue
		}
		if node.Type == wfv1.NodeTypePod && node.ID != "" {
			// Argo pod naming: <workflow>-<template>-<hash>; the node.ID is
			// the pod name for pod-type nodes in argo v3.
			podsToDelete = append(podsToDelete, node.ID)
		}
		node.Phase = wfv1.NodePending
		node.Message = ""
		node.FinishedAt = metav1.Time{}
		wf.Status.Nodes[id] = node
	}

	// Delete stale failed pods. Absent (NotFound) pods are fine — argo may
	// have GC'd them already.
	for _, pod := range podsToDelete {
		if err := c.pods.CoreV1().Pods(ns).Delete(ctx, pod, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("argo crd: delete pod %q for retry: %w", pod, err)
		}
	}
	if err := c.updateWorkflow(ctx, ns, wf); err != nil {
		return fmt.Errorf("argo crd: update workflow %q for retry: %w", name, err)
	}
	return nil
}

// ResubmitWorkflow clones the workflow spec into a new object and submits it
// with a fresh generateName. Hand-rolled equivalent of argo-server /resubmit
// (see RetryWorkflow comment for why not the upstream util). Simplification:
// no --memoized (previously-succeeded nodes are re-run).
func (c *crdWorkflowClient) ResubmitWorkflow(ctx context.Context, name, namespace string) error {
	_, err := c.ResubmitWorkflowWithResult(ctx, name, namespace)
	return err
}

// ResubmitWorkflowWithResult returns the newly created workflow so callers
// that need the fresh UID (Deploy usecase's re-submit-on-crash path) can
// capture it.
func (c *crdWorkflowClient) ResubmitWorkflowWithResult(ctx context.Context, name, namespace string) (*wfv1.Workflow, error) {
	ns := c.resolveNS(namespace)
	src, err := c.GetWorkflow(ctx, name, ns)
	if err != nil {
		return nil, err
	}

	// Deep-copy spec; clear metadata so the new object is a fresh submission,
	// not an update to the existing one.
	newWF := &wfv1.Workflow{}
	newWF.TypeMeta = src.TypeMeta
	newWF.Spec = *src.Spec.DeepCopy()
	newWF.ObjectMeta = metav1.ObjectMeta{
		Namespace:    src.Namespace,
		GenerateName: resubmitGenerateName(src),
		Labels:       resubmitLabels(src.Labels),
		Annotations:  resubmitAnnotations(src.Annotations),
	}

	u, err := workflowToUnstructured(newWF)
	if err != nil {
		return nil, err
	}
	created, err := c.dyn.Resource(workflowGVR).Namespace(ns).Create(ctx, u, metav1.CreateOptions{})
	if err != nil {
		return nil, translateK8sErr(err)
	}
	return unstructuredToWorkflow(created)
}

// updateWorkflow serializes a wfv1.Workflow back through the dynamic client.
// Used by RetryWorkflow; kept unexported so lifecycle ops (which use patch)
// don't accidentally take the heavier Update path.
func (c *crdWorkflowClient) updateWorkflow(ctx context.Context, ns string, wf *wfv1.Workflow) error {
	u, err := workflowToUnstructured(wf)
	if err != nil {
		return err
	}
	if _, err := c.dyn.Resource(workflowGVR).Namespace(ns).Update(ctx, u, metav1.UpdateOptions{}); err != nil {
		return translateK8sErr(err)
	}
	return nil
}

func isRetryable(p wfv1.WorkflowPhase) bool {
	return p == wfv1.WorkflowFailed || p == wfv1.WorkflowError || p == wfv1.WorkflowSucceeded
}

// resubmitGenerateName derives a fresh generateName from the source
// workflow's name / generateName so kubelet still assigns a unique name.
// Suffix is stripped of trailing hex to avoid ever-growing names on repeated
// resubmits.
func resubmitGenerateName(src *wfv1.Workflow) string {
	if src.GenerateName != "" {
		return src.GenerateName
	}
	// Strip argo's hash suffix (e.g. wf-abcd1) to reuse a stable prefix.
	base := src.Name
	if idx := strings.LastIndex(base, "-"); idx > 0 {
		base = base[:idx]
	}
	return base + "-"
}

// resubmitLabels copies user-defined labels but drops argo-managed ones that
// would collide when the workflow-controller starts assigning fresh values.
func resubmitLabels(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		if strings.HasPrefix(k, "workflows.argoproj.io/") {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// resubmitAnnotations copies annotations verbatim except a few argo-managed
// ones that shouldn't be inherited by the resubmitted workflow.
func resubmitAnnotations(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		if strings.HasPrefix(k, "workflows.argoproj.io/") {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}


// GetWorkflowLogs reads bounded logs for a workflow pod via the K8s pod log
// API. When podName is empty the caller wants every pod in the workflow
// interleaved by node startedAt (argo-server /log behavior); when podName is
// set only that pod's log is returned.
//
// Uses typed CoreV1().Pods().GetLogs — this is the same endpoint argo-server
// wraps internally, just skipping the argo-server hop.
func (c *crdWorkflowClient) GetWorkflowLogs(
	ctx context.Context,
	workflowName, podName, namespace string,
	opts WorkflowLogOptions,
) (WorkflowLogResult, error) {
	ns := c.resolveNS(namespace)
	pods, err := c.podsForLogs(ctx, workflowName, podName, ns)
	if err != nil {
		return WorkflowLogResult{}, err
	}
	logOpts := toPodLogOptions(opts)

	var out strings.Builder
	lineCount := 0
	for _, pod := range pods {
		raw, err := c.pods.CoreV1().Pods(ns).GetLogs(pod, &logOpts).DoRaw(ctx)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// Pod GC'd already — argo-server would return "" for missing
				// pods rather than erroring the whole log fetch. Mirror that.
				continue
			}
			return WorkflowLogResult{}, translateK8sErr(err)
		}
		out.Write(raw)
		if len(raw) > 0 && raw[len(raw)-1] != '\n' {
			out.WriteByte('\n')
		}
		lineCount += countLogLines(string(raw))
	}

	result := WorkflowLogResult{
		Logs:      out.String(),
		LineCount: lineCount,
	}
	if opts.LimitBytes != nil {
		limit := *opts.LimitBytes
		result.LimitBytes = limit
		if limit >= 0 && int64(len(result.Logs)) > limit {
			result.Logs = result.Logs[:limit]
			result.Truncated = true
			result.LineCount = countLogLines(result.Logs)
		}
	}
	return result, nil
}

// GetWorkflowLogStream returns a live log stream for a single pod. Multi-pod
// stream would require multiplexing separate log streams (argo-server does
// this) — for now the frontend already calls per-pod, so a single-pod stream
// covers the current UI. Multi-pod interleave streaming can land later.
func (c *crdWorkflowClient) GetWorkflowLogStream(
	ctx context.Context,
	workflowName, podName, namespace string,
	opts WorkflowLogOptions,
) (io.ReadCloser, error) {
	ns := c.resolveNS(namespace)
	if podName == "" {
		return nil, fmt.Errorf("argo crd: log stream requires an explicit pod name")
	}
	logOpts := toPodLogOptions(opts)
	stream, err := c.pods.CoreV1().Pods(ns).GetLogs(podName, &logOpts).Stream(ctx)
	if err != nil {
		return nil, translateK8sErr(err)
	}
	return stream, nil
}

// podLogEntry pairs a pod name with its startedAt for interleave sorting.
type podLogEntry struct {
	id    string
	start metav1.Time
}

// podsForLogs resolves the pod list for a log fetch:
//   - explicit podName → single-element list
//   - empty podName → enumerate workflow.status.nodes for Pod-type nodes,
//     sorted by node.StartedAt so multi-step logs interleave chronologically.
func (c *crdWorkflowClient) podsForLogs(ctx context.Context, workflowName, podName, ns string) ([]string, error) {
	if podName != "" {
		return []string{podName}, nil
	}
	wf, err := c.GetWorkflow(ctx, workflowName, ns)
	if err != nil {
		return nil, err
	}
	entries := make([]podLogEntry, 0, len(wf.Status.Nodes))
	for _, node := range wf.Status.Nodes {
		if node.Type != wfv1.NodeTypePod || node.ID == "" {
			continue
		}
		entries = append(entries, podLogEntry{id: node.ID, start: node.StartedAt})
	}
	if len(entries) > 1 {
		sortPodLogEntries(entries)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.id)
	}
	return out, nil
}

// sortPodLogEntries orders pods ascending by StartedAt (earliest first) so
// concatenated log output reads top-to-bottom in run order. Insertion sort:
// no `sort` import, and workflows are almost always <100 pods so O(n²) is
// irrelevant.
func sortPodLogEntries(entries []podLogEntry) {
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].start.Before(&entries[j-1].start); j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
}

// parseK8sTime parses the RFC3339 timestamp SinceTime uses.
func parseK8sTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// toPodLogOptions maps WorkflowLogOptions (argo-side) into K8s PodLogOptions.
// Container defaults to "main" — argo transpiler produces a single container
// per node with that name, matching argo-server's default too.
func toPodLogOptions(opts WorkflowLogOptions) corev1.PodLogOptions {
	container := opts.Container
	if container == "" {
		container = "main"
	}
	logOpts := corev1.PodLogOptions{
		Container:  container,
		Timestamps: opts.Timestamps,
		Follow:     opts.Follow,
		Previous:   opts.Previous,
	}
	if opts.TailLines != nil {
		logOpts.TailLines = opts.TailLines
	}
	if opts.LimitBytes != nil {
		logOpts.LimitBytes = opts.LimitBytes
	}
	if opts.SinceSeconds != nil {
		logOpts.SinceSeconds = opts.SinceSeconds
	}
	if opts.SinceTime != "" {
		if t, err := parseK8sTime(opts.SinceTime); err == nil {
			logOpts.SinceTime = &metav1.Time{Time: t}
		}
	}
	return logOpts
}

// resolveNS falls back to the cluster's default namespace when the caller
// passes an empty string. Consumers today always pass an explicit ns; keeping
// the fallback avoids nil-ns 400s if that convention slips.
func (c *crdWorkflowClient) resolveNS(ns string) string {
	if ns != "" {
		return ns
	}
	return c.defaultNS
}

// workflowToUnstructured encodes a wfv1.Workflow to an *unstructured.Unstructured
// through JSON marshalling. Doing it via JSON (not runtime.DefaultUnstructuredConverter)
// is deliberate: DefaultUnstructuredConverter walks via reflection and is
// known to mis-handle metav1.Time / metav1.Duration fields, dropping or
// mangling metadata.creationTimestamp, status.startedAt, status.finishedAt,
// and every status.nodes[].startedAt — precisely the fields the run watcher
// depends on to persist phase transitions.
//
// json.Marshal calls the type's own MarshalJSON (metav1.Time returns RFC3339;
// wfv1.NodeStatus keeps its shape), and unstructured.UnmarshalJSON accepts
// that as-is. Also normalizes TypeMeta so callers can hand in transpiler
// output without setting apiVersion/kind themselves.
func workflowToUnstructured(wf *wfv1.Workflow) (*unstructured.Unstructured, error) {
	if wf == nil {
		return nil, fmt.Errorf("argo crd: nil workflow")
	}
	// unstructured.UnmarshalJSON requires apiVersion + kind in the payload.
	// Shallow-copy the workflow so we can normalize TypeMeta without mutating
	// the caller's pointer (Spec / ObjectMeta stay shared — we only replace
	// the TypeMeta value field).
	withGVK := *wf
	withGVK.TypeMeta = metav1.TypeMeta{
		APIVersion: "argoproj.io/v1alpha1",
		Kind:       "Workflow",
	}
	data, err := json.Marshal(&withGVK)
	if err != nil {
		return nil, fmt.Errorf("argo crd: encode workflow: %w", err)
	}
	u := &unstructured.Unstructured{}
	if err := u.UnmarshalJSON(data); err != nil {
		return nil, fmt.Errorf("argo crd: encode workflow (unstructured): %w", err)
	}
	return u, nil
}

// unstructuredToWorkflow decodes an unstructured Workflow CRD into wfv1
// through JSON. See workflowToUnstructured for why this doesn't use
// runtime.DefaultUnstructuredConverter (metav1.Time reflection bugs).
// Failure to decode is a hard error — the alternative (returning a partial
// object) would silently drop fields callers depend on.
func unstructuredToWorkflow(u *unstructured.Unstructured) (*wfv1.Workflow, error) {
	if u == nil {
		return nil, fmt.Errorf("argo crd: nil workflow unstructured")
	}
	data, err := u.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("argo crd: encode unstructured %q: %w", u.GetName(), err)
	}
	wf := &wfv1.Workflow{}
	if err := json.Unmarshal(data, wf); err != nil {
		return nil, fmt.Errorf("argo crd: decode workflow %q: %w", u.GetName(), err)
	}
	return wf, nil
}

// translateK8sErr maps K8s API errors into argo package sentinels the rest of
// the backend already knows how to handle. IsNotFound → ErrNotFound;
// IsAlreadyExists → ErrAlreadyExists; everything else passes through.
func translateK8sErr(err error) error {
	if err == nil {
		return nil
	}
	if apierrors.IsNotFound(err) {
		return fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	if apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("%w: %v", ErrAlreadyExists, err)
	}
	return err
}
