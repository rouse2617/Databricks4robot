# Task 🅱️: Pipeline usecase + transpiler 轻量化

## 目标
去掉 `pipeline usecase` 对 `corev1.EnvVar` 的依赖（仅剩的运行时 K8s 类型引用），改为内部轻量类型。这样 usecase 不再需要 import `k8s.io/api/core/v1`。

注意：transpiler 自身仍需要用 corev1/wfv1 类型构建 Argo Workflow JSON（wfv1.Template.Container = *corev1.Container），这不影响。

## 具体改动

### 1. `backend/internal/transpiler/transpiler.go`
- 新增内部类型：
```go
// EnvVar represents a name/value pair for environment variables.
type EnvVar struct {
    Name  string
    Value string
}

// Volume represents a named volume that can be mounted.
type Volume struct {
    Name       string
    IsEmptyDir bool
    PVCName    string
}
```

- `Options.GlobalEnv []corev1.EnvVar` → `Options.GlobalEnv []EnvVar`
- `Options.ExtraVolumes []corev1.Volume` → `Options.ExtraVolumes []Volume`
- `buildContainerTemplate` 中：在构建 `tmpl.Container.Env` 时，把 `opts.GlobalEnv` 转成 `corev1.EnvVar`（用内部类型转）
- `buildWorkflowVolumes` 中：把 `[]Volume` 转成 `[]corev1.Volume`（保持内部逻辑不变）

### 2. `backend/internal/usecase/pipeline/usecase.go`
- 去掉 import `corev1 "k8s.io/api/core/v1"`
- 所有 `corev1.EnvVar{Name: ..., Value: ...}` 改为 `transpiler.EnvVar{Name: ..., Value: ...}`
- 验证：`go build ./...`

### 3. 清理 `usecase.go` 中 `corev1` import
确保 `go build ./...` 通过，不再引用 `k8s.io/api/core/v1` 等运行时依赖。

## 约束
- 不改 transpiler 的 wfv1/corev1 导入（transpiler 需要这些构建 Argo Workflow JSON）
- 不改任何 test 文件
- 不改其他包
- 验证：`cd backend && go build ./... && go vet ./internal/usecase/pipeline/...`
