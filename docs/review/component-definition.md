# Component Definition Design: Container vs Script vs Docker Image

> **Date:** 2026-05-28
> **Status:** Draft
> **Author:** Hermes

## Background

cyber-databrew's pipeline system lets users register components and compose them into DAG workflows. Each component is transpiled into an Argo Workflows template.

### Current Architecture

The [transpiler](/Users/rick/cyber-databrew/backend/internal/transpiler/transpiler.go:263-398) always emits **`container` template** type:

```go
// Component model (pipeline.go:26-34)
type Component struct {
    Image           string      // container image
    Command         []string    // entrypoint
    Args            []Argument  // positional args (Name/Value/From)
    Env             []EnvVar
    Resources       *ResourceRequirements
}
```

Output parameters are declared via `valueFrom.path: /tmp/outputs/<name>` (transpiler.go:325), but **no code creates this directory inside the container**. This causes the E2E error:

```
/bin/sh: line 0: can't create /tmp/outputs/output: nonexistent directory
```

### The Core Problem

The transpiler declares output file paths but provides no mechanism to ensure the parent directory exists. Neither Argo's `container` template nor its `script` template automatically creates `/tmp/outputs/` — it is always the user's (or the transpiler's) responsibility.

---

## 1. Approach Comparison

### Approach A: Container + Inline Args (Current)

```yaml
# transpiler output
- name: step-node-1
  container:
    image: busybox
    command: [sh, -c]
    args: ["echo 'hello' > /tmp/outputs/output"]
  outputs:
    parameters:
      - name: output
        valueFrom:
          path: /tmp/outputs/output
```

| Aspect | Rating | Notes |
|--------|--------|-------|
| **Simplicity** | ⭐⭐⭐⭐⭐ | Straightforward, minimal code |
| **Multi-line scripts** | ⭐⭐ | Must inline into `-c` arg; escaping hell |
| **Debugging** | ⭐⭐ | No source annotation in workflow YAML |
| **Output path mgmt** | ⭐ | Directory must be created manually |
| **Argo integration** | ⭐⭐⭐ | Works with all Argo features |
| **Flexibility** | ⭐⭐⭐⭐ | Can run any container image |

### Approach B: Argo Script Template

```yaml
- name: step-node-1
  script:
    image: busybox
    command: [sh]
    source: |
      echo 'hello' > /tmp/outputs/output
  outputs:
    parameters:
      - name: output
        valueFrom:
          path: /tmp/outputs/output
```

Argo writes `source` to a temporary file and executes `command < tmpfile`. Stdout is automatically captured as `outputs.result`.

| Aspect | Rating | Notes |
|--------|--------|-------|
| **Simplicity** | ⭐⭐⭐⭐ | Clean YAML, no escaping |
| **Multi-line scripts** | ⭐⭐⭐⭐⭐ | Native support via `source:` |
| **Debugging** | ⭐⭐⭐⭐ | Script visible in Argo UI |
| **Output path mgmt** | ⭐⭐ | Argo does NOT auto-create `/tmp/outputs/` |
| **Argo integration** | ⭐⭐⭐⭐⭐ | First-class Argo feature |
| **Flexibility** | ⭐⭐⭐ | Requires `command` to be an interpreter |

### Approach C: Docker Image Mode (Kubeflow-style)

```yaml
- name: step-node-1
  container:
    image: gcr.io/my-org/my-component:v1
    command: [python3, /program.py]
    args: ["--input", "{{inputs.parameters.input}}", "--output", "/tmp/outputs/output"]
```

Component = opaque Docker image. Platform only passes params and mounts volumes. Image manages its own filesystem.

| Aspect | Rating | Notes |
|--------|--------|-------|
| **Simplicity** | ⭐⭐⭐ | Requires building and pushing images |
| **Multi-line scripts** | ⭐⭐⭐⭐⭐ | Trivial (code is in the image) |
| **Debugging** | ⭐⭐⭐⭐⭐ | Fully testable offline |
| **Output path mgmt** | ⭐⭐⭐⭐ | Image handles it |
| **Argo integration** | ⭐⭐⭐ | Standard container template |
| **Flexibility** | ⭐⭐⭐⭐⭐ | Any language, any dependencies |
| **Onboarding friction** | ⭐⭐ | Need Docker + registry account |

---

## 2. Industry Comparison

| Feature | cyber-databrew (current) | Argo Workflows | Kubeflow Pipelines v2 | Flyte | Airflow |
|---------|-------------------------|----------------|----------------------|-------|---------|
| **Component spec** | Go struct | Template types | `component.yaml` | Task decorator | Operator class |
| **Execution model** | Container template | Container / Script / DAG | Container executor | Pod / Container | Worker pod |
| **Output mechanism** | `valueFrom.path` | `valueFrom.path` / stdout | `outputPath` placeholder | `@outputs` annotation | XCom |
| **Auto-creates output dir?** | ❌ | ❌ (both script & container) | ✅ (KFP executor) | ✅ (Flyte propeller) | ❌ (manual) |
| **Inline script support** | Via `-c` args | ✅ `source:` field | Via Python functions | Via `@task` | Via `BashOperator` |
| **Multi-language** | ✅ (any image) | ✅ (any image) | ✅ (any image) | ✅ (any image) | Python-only |
| **Output dir convention** | `/tmp/outputs/` | User-defined | `/tmp/outputs/` (KFP v2) | User-defined | User-defined |

### Key Insight

**Kubeflow** is the only platform that auto-creates output directories. It does so through its own executor binary injected into each step, not through the workflow engine. Neither Argo's `script` nor `container` templates provide this.

---

## 3. Recommended Approach

### Decision

**Approach A+B (hybrid)** — support both `container` and `script` template modes:

| Mode | Template Type | Use Case |
|------|---------------|----------|
| `container` (default) | Argo `container` | Production components, pre-built images, `mkdir -p` auto-injected |
| `script` | Argo `script` | Quick prototyping, inline scripts, multi-line logic |

### Rationale

1. **Backward compatible** — `container` mode with auto `mkdir -p` fixes the E2E bug without breaking existing components
2. **Low onboarding friction** — `script` mode lets users write inline scripts without Docker
3. **Production-ready** — `container` mode supports any image for serious workloads
4. **Docker Image mode deferred** — not yet needed; revisit when we see demand for shared component registries

### Why NOT Docker Image mode now

- Requires users to build and push images — high friction for a platform in early adoption
- Adds CI/CD complexity (image registry, build pipeline, versioning)
- Can be layered on top of `container` mode later without breaking changes
- The current `container` mode already runs arbitrary images; "Docker Image mode" just means the component *is* the image with no inline script

---

## 4. Implementation

### 4.1 Data Model Changes

**File:** `backend/internal/transpiler/pipeline.go`

Add to `Component` struct:

```go
// Component is a pipeline step backed by a container image.
type Component struct {
    Name            string                `json:"name" yaml:"name"`
    Image           string                `json:"image" yaml:"image"`
    ImagePullPolicy string                `json:"imagePullPolicy,omitempty" yaml:"imagePullPolicy,omitempty"`
    Command         []string              `json:"command,omitempty" yaml:"command,omitempty"`
    Args            []Argument            `json:"args,omitempty" yaml:"args,omitempty"`
    
    // NEW FIELD
    Mode   string   `json:"mode,omitempty" yaml:"mode,omitempty"`     // "container" (default) or "script"
    Source string   `json:"source,omitempty" yaml:"source,omitempty"`  // inline script body (script mode only)
    
    Env             []EnvVar              `json:"env,omitempty" yaml:"env,omitempty"`
    Resources       *ResourceRequirements `json:"resources,omitempty" yaml:"resources,omitempty"`
}
```

### 4.2 Transpiler Changes

**File:** `backend/internal/transpiler/transpiler.go`

Three changes:

#### a) Dispatch in `buildNodeTemplates` (line 465–469)

```go
func buildNodeTemplates(node Node, inputs []inputSpec, opts *Options) ([]wfv1.Template, error) {
    if len(node.SubNodes) > 0 {
        return buildSubGraphTemplates(node, inputs, opts)
    }
    if node.Component.Mode == "script" {
        return []wfv1.Template{*buildScriptTemplate(node, inputs, opts)}, nil
    }
    return []wfv1.Template{*buildContainerTemplate(node, inputs, opts)}, nil
}
```

#### b) Inject `mkdir -p` in `buildContainerTemplate` (line 332–343)

```go
var containerArgs []string

// Auto-create output directory when component declares outputs
if len(node.Outputs) > 0 {
    containerArgs = append(containerArgs, "mkdir", "-p", "/tmp/outputs", "&&")
}

for _, arg := range node.Component.Args {
    // ... existing arg building logic ...
}
```

**Important:** This only works when `command` is `["sh", "-c"]`. For arbitrary commands, we must not inject shell syntax. We'll check: if the command is `["sh", "-c"]`, prepend `mkdir -p /tmp/outputs &&` to the single `-c` argument; otherwise, skip the auto-injection.

#### c) New `buildScriptTemplate` function

```go
func buildScriptTemplate(node Node, inputs []inputSpec, opts *Options) *wfv1.Template {
    // Start with a copy of container template setup (image, pull policy, resources, env, volumes)
    tmpl := buildContainerTemplate(node, inputs, opts)
    
    // Convert from container to script template
    script := &wfv1.ScriptTemplate{
        Image:           tmpl.Container.Image,
        Command:         tmpl.Container.Command,
        ImagePullPolicy: tmpl.Container.ImagePullPolicy,
        Source:          node.Component.Source,
    }
    
    if len(script.Command) == 0 {
        script.Command = []string{"sh"}  // default interpreter
    }
    
    if script.Source == "" {
        // Fallback: build source from Args
        script.Source = buildSourceFromArgs(node)
    }
    
    // Prepend mkdir -p to source when outputs are declared
    if len(node.Outputs) > 0 {
        script.Source = "mkdir -p /tmp/outputs\n" + script.Source
    }
    
    tmpl.Container = nil
    tmpl.Script = script
    return tmpl
}
```

### 4.3 Backward Compatibility

- `Mode` defaults to `""` (empty string), treated as `"container"`
- Existing components with `Args` continue to work unchanged
- `mkdir -p` is only injected when `Outputs` are declared AND command is `sh -c`
- `Source` field is ignored unless `Mode == "script"`

### 4.4 Frontend Integration

The frontend component editor should expose a toggle:

- **Container mode** (default): show Image, Command, Args, Env fields
- **Script mode**: show Image, Script editor (monaco/codemirror), Env fields; Args are auto-derived from the Source

Frontend changes are not part of this phase.

---

## 5. Future Considerations

| Phase | Scope | When |
|-------|-------|------|
| 1 | `mkdir -p` quick fix + `buildScriptTemplate` | Now |
| 2 | Frontend script editor, Argo UI source preview | Next sprint |
| 3 | Docker Image mode (component-as-image registry) | Backlog |
| 4 | KFP-compatible `component.yaml` import | Backlog |

---

## 6. References

- [Argo Script Template Docs](https://argo-workflows.readthedocs.io/en/latest/script-template/)
- [Argo Output Parameters](https://argo-workflows.readthedocs.io/en/latest/walk-through/output-parameters/)
- [Kubeflow Component Spec](https://www.kubeflow.org/docs/components/pipelines/reference/component-spec/)
- [Transpiler source](/Users/rick/cyber-databrew/backend/internal/transpiler/transpiler.go)
- [Data model](/Users/rick/cyber-databrew/backend/internal/transpiler/pipeline.go)
- [Argo integration research](argo-integration-research.md)
