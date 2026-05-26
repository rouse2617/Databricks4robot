package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cyber-databrew/databrew-pipeline/transpiler"
	"sigs.k8s.io/yaml"
)

var (
	dataDir     string
	kubeconfig  string
	wfNamespace string
)

func init() {
	home, _ := os.UserHomeDir()
	dataDir = filepath.Join(home, ".databrew")
	os.MkdirAll(filepath.Join(dataDir, "pipelines"), 0755)
	os.MkdirAll(filepath.Join(dataDir, "deployments"), 0755)

	if kc := os.Getenv("KUBECONFIG"); kc != "" {
		kubeconfig = kc
	} else if _, err := os.Stat("/tmp/kubeconfig-argo-local"); err == nil {
		kubeconfig = "/tmp/kubeconfig-argo-local"
	} else {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	wfNamespace = os.Getenv("WF_NS")
	if wfNamespace == "" {
		// Local kind cluster (kubeconfig-argo-local) uses sandbox-project-a; GKE dev uses cyber-databrew-dev.
		if strings.Contains(kubeconfig, "kubeconfig-argo-local") {
			wfNamespace = "sandbox-project-a"
		} else {
			wfNamespace = "cyber-databrew-dev"
		}
	}
}

// ── Models ──────────────────────────────────────────────────────

type PipelineTemplate struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Pipeline  json.RawMessage `json:"pipeline"`
	NodeCount int             `json:"nodeCount"`
	CreatedAt string          `json:"createdAt"`
}

type DeployRequest struct {
	Pipeline   json.RawMessage `json:"pipeline"`
	TemplateID string          `json:"templateId,omitempty"`
	Name       string          `json:"name,omitempty"`
}

type DeploymentRecord struct {
	ID           string `json:"id"`
	PipelineName string `json:"pipelineName"`
	WorkflowName string `json:"workflowName"`
	Status       string `json:"status"`
	Nodes        int    `json:"nodes"`
	CreatedAt    string `json:"createdAt"`
	FinishedAt   string `json:"finishedAt,omitempty"`
	Manifest     string `json:"manifest,omitempty"`
	PipelineJSON string `json:"pipelineJSON,omitempty"`
}

type APIResponse struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

// ── Helpers ─────────────────────────────────────────────────────

func genID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, APIResponse{Error: msg})
}

func kubectl(args ...string) (string, string, error) {
	cmd := exec.Command("kubectl", append([]string{
		"--kubeconfig", kubeconfig,
		"-n", wfNamespace,
	}, args...)...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func kubectlApply(manifest string) (string, error) {
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig, "-n", wfNamespace, "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(manifest)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func refreshStatus(wfName string) string {
	out, stderr, err := kubectl("get", "wf", wfName, "-o", "jsonpath={.status.phase}")
	if err != nil {
		msg := stderr + out
		if strings.Contains(msg, "NotFound") || strings.Contains(msg, "not found") {
			return "NotFound"
		}
		return "Unknown"
	}
	phase := strings.TrimSpace(out)
	if phase == "" {
		return "Pending"
	}
	return phase
}

func workflowExists(wfName string) bool {
	out, _, err := kubectl("get", "wf", wfName, "-o", "name")
	return err == nil && strings.TrimSpace(out) != ""
}

func nowStr() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// ── Pipeline Templates ──────────────────────────────────────────

func handleSavePipeline(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, 400, "cannot read body")
		return
	}

	var req struct {
		Name     string          `json:"name"`
		Pipeline json.RawMessage `json:"pipeline"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if req.Name == "" || len(req.Pipeline) == 0 {
		writeError(w, 400, "name and pipeline required")
		return
	}

	// Count nodes
	var raw struct {
		Nodes []interface{} `json:"nodes"`
	}
	json.Unmarshal(req.Pipeline, &raw)
	nodeCount := len(raw.Nodes)

	tpl := PipelineTemplate{
		ID:        genID(),
		Name:      req.Name,
		Pipeline:  req.Pipeline,
		NodeCount: nodeCount,
		CreatedAt: nowStr(),
	}

	data, _ := json.Marshal(tpl)
	os.WriteFile(filepath.Join(dataDir, "pipelines", tpl.ID+".json"), data, 0644)

	writeJSON(w, 200, APIResponse{OK: true, Data: tpl})
}

func handleListPipelines(w http.ResponseWriter, r *http.Request) {
	files, _ := filepath.Glob(filepath.Join(dataDir, "pipelines", "*.json"))
	var list []PipelineTemplate
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var tpl PipelineTemplate
		if json.Unmarshal(data, &tpl) == nil {
			list = append(list, tpl)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt > list[j].CreatedAt })
	writeJSON(w, 200, APIResponse{OK: true, Data: list})
}

func handleDeletePipeline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, 400, "id required")
		return
	}
	path := filepath.Join(dataDir, "pipelines", id+".json")
	os.Remove(path)
	writeJSON(w, 200, APIResponse{OK: true})
}

func handleGetPipeline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, 400, "id required")
		return
	}
	data, err := os.ReadFile(filepath.Join(dataDir, "pipelines", id+".json"))
	if err != nil {
		writeError(w, 404, "pipeline not found")
		return
	}
	var tpl PipelineTemplate
	if json.Unmarshal(data, &tpl) != nil {
		writeError(w, 500, "corrupt pipeline file")
		return
	}
	writeJSON(w, 200, APIResponse{OK: true, Data: tpl})
}

// ── Deploy ──────────────────────────────────────────────────────

func handleDeploy(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, 400, "cannot read body")
		return
	}

	var req DeployRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}

	// Resolve pipeline JSON: inline or from template
	pipeData := req.Pipeline
	if req.TemplateID != "" {
		tplData, err := os.ReadFile(filepath.Join(dataDir, "pipelines", req.TemplateID+".json"))
		if err != nil {
			writeError(w, 404, "template not found")
			return
		}
		var tpl PipelineTemplate
		if json.Unmarshal(tplData, &tpl) != nil {
			writeError(w, 500, "corrupt template")
			return
		}
		pipeData = tpl.Pipeline
		req.Name = tpl.Name
	}

	var pipe transpiler.Pipeline
	if err := json.Unmarshal(pipeData, &pipe); err != nil {
		writeError(w, 400, "invalid pipeline JSON: "+err.Error())
		return
	}

	// Generate a unique workflow name
	wfName := pipe.Name
	if req.Name != "" {
		wfName = req.Name
	}
	suffix := genID()[:6]
	wfName = wfName + "-" + suffix

	// Transpile to Argo Workflow
	opts := &transpiler.Options{
		Name:            wfName,
		Namespace:       wfNamespace,
		TTLSecondsAfter: 3600,
	}
	wf, err := transpiler.Transpile(&pipe, opts)
	if err != nil {
		writeError(w, 500, "transpile error: "+err.Error())
		return
	}

	// Marshal to JSON first, then convert to YAML (preserves correct field names)
	wfJSON, err := json.Marshal(wf)
	if err != nil {
		writeError(w, 500, "json marshal error: "+err.Error())
		return
	}
	manifest, err := yaml.JSONToYAML(wfJSON)
	if err != nil {
		writeError(w, 500, "yaml marshal error: "+err.Error())
		return
	}

	applyOut, applyErr := kubectlApply(string(manifest))
	if applyErr != nil {
		writeError(w, 500, fmt.Sprintf("kubectl apply failed (namespace=%s): %s\n%s", wfNamespace, applyOut, applyErr.Error()))
		return
	}

	initialStatus := refreshStatus(wfName)
	if initialStatus == "Unknown" || initialStatus == "NotFound" {
		if !workflowExists(wfName) {
			writeError(w, 500, fmt.Sprintf("workflow %q not found after apply in namespace %s (is Argo Workflows installed? set WF_NS). kubectl: %s", wfName, wfNamespace, strings.TrimSpace(applyOut)))
			return
		}
		initialStatus = "Pending"
	}

	// Save deployment record
	deployment := DeploymentRecord{
		ID:           genID(),
		PipelineName: pipe.Name,
		WorkflowName: wfName,
		Status:       initialStatus,
		Nodes:        len(pipe.Nodes),
		CreatedAt:    nowStr(),
		Manifest:     string(manifest),
		PipelineJSON: string(pipeData),
	}
	depData, _ := json.Marshal(deployment)
	os.WriteFile(filepath.Join(dataDir, "deployments", deployment.ID+".json"), depData, 0644)

	writeJSON(w, 200, APIResponse{OK: true, Data: deployment})
}

func handleListDeployments(w http.ResponseWriter, r *http.Request) {
	files, _ := filepath.Glob(filepath.Join(dataDir, "deployments", "*.json"))
	var list []DeploymentRecord
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var dep DeploymentRecord
		if json.Unmarshal(data, &dep) == nil {
			// Refresh status from cluster
			if dep.Status == "Running" || dep.Status == "Pending" || dep.Status == "Unknown" {
				dep.Status = refreshStatus(dep.WorkflowName)
				if dep.Status == "Succeeded" || dep.Status == "Failed" {
					dep.FinishedAt = nowStr()
				}
				upd, _ := json.Marshal(dep)
				os.WriteFile(f, upd, 0644)
			}
			list = append(list, dep)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt > list[j].CreatedAt })
	writeJSON(w, 200, APIResponse{OK: true, Data: list})
}

func handleGetDeployment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, 400, "id required")
		return
	}
	data, err := os.ReadFile(filepath.Join(dataDir, "deployments", id+".json"))
	if err != nil {
		writeError(w, 404, "deployment not found")
		return
	}
	var dep DeploymentRecord
	json.Unmarshal(data, &dep)

	// Refresh status
	dep.Status = refreshStatus(dep.WorkflowName)
	if dep.Status == "Succeeded" || dep.Status == "Failed" || dep.Status == "Error" {
		dep.FinishedAt = nowStr()
	}
	upd, _ := json.Marshal(dep)
	os.WriteFile(filepath.Join(dataDir, "deployments", id+".json"), upd, 0644)

	writeJSON(w, 200, APIResponse{OK: true, Data: dep})
}

func handleDeleteDeployment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, 400, "id required")
		return
	}
	// Read record to get workflow name
	data, err := os.ReadFile(filepath.Join(dataDir, "deployments", id+".json"))
	if err == nil {
		var dep DeploymentRecord
		json.Unmarshal(data, &dep)
		// Delete the workflow from cluster (ignore error)
		kubectl("delete", "wf", dep.WorkflowName, "--ignore-not-found")
	}
	os.Remove(filepath.Join(dataDir, "deployments", id+".json"))
	writeJSON(w, 200, APIResponse{OK: true})
}

// ── Middleware ───────────────────────────────────────────────────

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ── Main ────────────────────────────────────────────────────────

func main() {
	mux := http.NewServeMux()

	// Pipeline templates
	mux.HandleFunc("GET /api/pipelines", handleListPipelines)
	mux.HandleFunc("POST /api/pipelines", handleSavePipeline)
	mux.HandleFunc("DELETE /api/pipelines/{id}", handleDeletePipeline)

	// Single pipeline (use /pipeline/ singular to avoid conflict with /api/pipelines)
	mux.HandleFunc("GET /api/pipeline/{id}", handleGetPipeline)
	mux.HandleFunc("POST /api/deploy", handleDeploy)
	mux.HandleFunc("GET /api/deployments", handleListDeployments)
	mux.HandleFunc("GET /api/deployments/{id}", handleGetDeployment)
	mux.HandleFunc("DELETE /api/deployments/{id}", handleDeleteDeployment)

	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("DataBrew API server starting on %s", addr)
	log.Printf("  Kubeconfig: %s", kubeconfig)
	log.Printf("  Workflow NS: %s", wfNamespace)
	log.Printf("  Storage: %s", dataDir)

	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
