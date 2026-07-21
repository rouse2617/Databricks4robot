package transpiler

import (
	"strings"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestTranspileEdgePorts(t *testing.T) {
	p := &Pipeline{
		Name: "test",
		Nodes: []Node{
			{
				ID: "a",
				Component: Component{
					Name:    "a",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo a > /tmp/outputs/out"}},
				},
				Outputs: []Port{{Name: "out", Type: "string"}},
			},
			{
				ID: "b",
				Component: Component{
					Name:    "b",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo {{inputs.parameters.in}}"}},
				},
				Inputs: []Port{{Name: "in", Type: "string"}},
			},
		},
		Edges: []Edge{{Source: "a.out", Target: "b.in"}},
	}

	wf, err := Transpile(p, &Options{Name: "wf-test", Namespace: "default"})
	if err != nil {
		t.Fatal(err)
	}
	if wf.Spec.Entrypoint != "dag" {
		t.Fatalf("entrypoint = %q", wf.Spec.Entrypoint)
	}
}

func TestTranspileRejectsDuplicateTargetInputs(t *testing.T) {
	p := &Pipeline{
		Name: "dup-target",
		Nodes: []Node{
			{
				ID: "a",
				Component: Component{
					Name:    "a",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo a > /tmp/outputs/out"}},
				},
				Outputs: []Port{{Name: "out", Type: "string"}},
			},
			{
				ID: "b",
				Component: Component{
					Name:    "b",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo b > /tmp/outputs/out"}},
				},
				Outputs: []Port{{Name: "out", Type: "string"}},
			},
			{
				ID: "join",
				Component: Component{
					Name:    "join",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo join"}},
				},
				Inputs: []Port{{Name: "input", Type: "string"}},
			},
		},
		Edges: []Edge{
			{Source: "a.out", Target: "join.input"},
			{Source: "b.out", Target: "join.input"},
		},
	}

	_, err := Transpile(p, &Options{Name: "dup-target"})
	if err == nil {
		t.Fatal("expected duplicate target input error")
	}
	if !strings.Contains(err.Error(), "join.input") {
		t.Fatalf("error = %q, want join.input detail", err.Error())
	}
}

func TestTranspileAllowsDistinctFanInInputs(t *testing.T) {
	p := &Pipeline{
		Name: "distinct-target",
		Nodes: []Node{
			{
				ID: "left",
				Component: Component{
					Name:    "left",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo left > /tmp/outputs/out"}},
				},
				Outputs: []Port{{Name: "out", Type: "string"}},
			},
			{
				ID: "right",
				Component: Component{
					Name:    "right",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo right > /tmp/outputs/out"}},
				},
				Outputs: []Port{{Name: "out", Type: "string"}},
			},
			{
				ID: "join",
				Component: Component{
					Name:    "join",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo join"}},
				},
				Inputs: []Port{{Name: "left", Type: "string"}, {Name: "right", Type: "string"}},
			},
		},
		Edges: []Edge{
			{Source: "left.out", Target: "join.left"},
			{Source: "right.out", Target: "join.right"},
		},
	}

	if _, err := Transpile(p, &Options{Name: "distinct-target"}); err != nil {
		t.Fatal(err)
	}
}

func TestTranspileEmitsGPUResourceLimit(t *testing.T) {
	p := &Pipeline{
		Name: "gpu-pipeline",
		Nodes: []Node{{
			ID: "gpu-step",
			Component: Component{
				Name:    "gpu",
				Image:   "nvidia/cuda:12.4.1-base-ubuntu22.04",
				Command: []string{"sh", "-c"},
				Args:    []Argument{{Name: "script", Value: "nvidia-smi"}},
				Resources: &ResourceRequirements{
					CPU:         "4000m",
					Memory:      "16Gi",
					Disk:        "50Gi",
					GPU:         "1",
					ComputeTier: "gpu-l4",
				},
			},
		}},
	}

	wf, err := Transpile(p, &Options{Name: "gpu-pipeline"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name != "step-gpu" {
			continue
		}
		if tmpl.Container == nil {
			t.Fatal("expected container template")
		}
		gpu := tmpl.Container.Resources.Limits[corev1.ResourceName("nvidia.com/gpu")]
		if gpu.String() != "1" {
			t.Fatalf("gpu limit = %q, want 1", gpu.String())
		}
		if _, ok := tmpl.Container.Resources.Requests[corev1.ResourceName("nvidia.com/gpu")]; ok {
			t.Fatal("gpu must not be emitted as a request")
		}
		if got := tmpl.NodeSelector["cloud.google.com/gke-accelerator"]; got != "nvidia-l4" {
			t.Fatalf("gpu node selector = %q, want nvidia-l4", got)
		}
		assertTemplateToleration(t, tmpl, "nvidia.com/gpu", "present")
		disk := tmpl.Container.Resources.Limits[corev1.ResourceEphemeralStorage]
		if got := disk.String(); got != "50Gi" {
			t.Fatalf("disk limit = %q, want 50Gi", got)
		}
		return
	}
	t.Fatal("step-gpu template not found")
}

func assertTemplateToleration(t *testing.T, tmpl wfv1.Template, key, value string) {
	t.Helper()
	for _, tol := range tmpl.Tolerations {
		if tol.Key == key && tol.Value == value && tol.Operator == corev1.TolerationOpEqual && tol.Effect == corev1.TaintEffectNoSchedule {
			return
		}
	}
	t.Fatalf("missing toleration %s=%s in %#v", key, value, tmpl.Tolerations)
}

// GpuStepNodeSelector must merge onto a GPU step's NodeSelector alongside the
// auto-added GKE accelerator hint, giving the pool the ability to pin its GPU
// workload to a specific node pool.
func TestTranspileAppliesGpuStepNodeSelectorToGpuStep(t *testing.T) {
	p := &Pipeline{
		Name: "gpu-pin",
		Nodes: []Node{{
			ID: "gpu-step",
			Component: Component{
				Name:  "gpu",
				Image: "nvidia/cuda:12.4.1-base-ubuntu22.04",
				Resources: &ResourceRequirements{
					CPU: "4000m", Memory: "16Gi", GPU: "1", ComputeTier: "gpu-l4",
				},
			},
		}},
	}
	wf, err := Transpile(p, &Options{
		Name: "gpu-pin",
		GpuStepNodeSelector: map[string]string{
			"cloud.google.com/gke-nodepool": "g2-l4-dev-pool",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name != "step-gpu" {
			continue
		}
		if got := tmpl.NodeSelector["cloud.google.com/gke-nodepool"]; got != "g2-l4-dev-pool" {
			t.Fatalf("gke-nodepool selector = %q, want g2-l4-dev-pool", got)
		}
		// The GKE accelerator hint must still be there (GPU-only auto path).
		if got := tmpl.NodeSelector["cloud.google.com/gke-accelerator"]; got != "nvidia-l4" {
			t.Fatalf("gke-accelerator selector = %q, want nvidia-l4", got)
		}
		return
	}
	t.Fatal("step-gpu template not found")
}

// GpuStepNodeSelector must NOT leak onto CPU steps — that's the whole point of
// the separate field. If it did, a mixed GPU+CPU pipeline's CPU pods would
// inherit a GPU-only pool label and get rejected by that pool's GPU taint.
func TestTranspileDoesNotApplyGpuStepNodeSelectorToCpuStep(t *testing.T) {
	p := &Pipeline{
		Name: "mixed",
		Nodes: []Node{
			{
				ID: "cpu-step",
				Component: Component{
					Name:  "cpu",
					Image: "busybox:latest",
					Resources: &ResourceRequirements{
						CPU: "2000m", Memory: "4Gi",
					},
				},
			},
			{
				ID: "gpu-step",
				Component: Component{
					Name:  "gpu",
					Image: "nvidia/cuda:12.4.1-base-ubuntu22.04",
					Resources: &ResourceRequirements{
						CPU: "4000m", Memory: "16Gi", GPU: "1", ComputeTier: "gpu-l4",
					},
				},
			},
		},
	}
	wf, err := Transpile(p, &Options{
		Name: "mixed",
		GpuStepNodeSelector: map[string]string{
			"cloud.google.com/gke-nodepool": "g2-l4-dev-pool",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var sawCPU, sawGPU bool
	for _, tmpl := range wf.Spec.Templates {
		switch tmpl.Name {
		case "step-cpu":
			sawCPU = true
			if _, ok := tmpl.NodeSelector["cloud.google.com/gke-nodepool"]; ok {
				t.Fatalf("CPU step must not inherit GpuStepNodeSelector; got %#v", tmpl.NodeSelector)
			}
		case "step-gpu":
			sawGPU = true
			if got := tmpl.NodeSelector["cloud.google.com/gke-nodepool"]; got != "g2-l4-dev-pool" {
				t.Fatalf("GPU step gke-nodepool selector = %q, want g2-l4-dev-pool", got)
			}
		}
	}
	if !sawCPU || !sawGPU {
		t.Fatalf("expected both step-cpu and step-gpu templates; sawCPU=%v sawGPU=%v", sawCPU, sawGPU)
	}
}

// A GpuStepNodeSelector entry with an empty key or value must be silently
// dropped (defensive; mirrors the TemplateNodeSelector guard).
func TestTranspileGpuStepNodeSelectorSkipsEmptyEntries(t *testing.T) {
	p := &Pipeline{
		Name: "gpu-empty",
		Nodes: []Node{{
			ID: "gpu-step",
			Component: Component{
				Name:  "gpu",
				Image: "nvidia/cuda:12.4.1-base-ubuntu22.04",
				Resources: &ResourceRequirements{
					CPU: "4000m", Memory: "16Gi", GPU: "1", ComputeTier: "gpu-l4",
				},
			},
		}},
	}
	wf, err := Transpile(p, &Options{
		Name: "gpu-empty",
		GpuStepNodeSelector: map[string]string{
			"":                              "ignored",
			"cloud.google.com/gke-nodepool": "",
			"real":                          "value",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name != "step-gpu" {
			continue
		}
		if got := tmpl.NodeSelector["real"]; got != "value" {
			t.Fatalf("real selector = %q, want value", got)
		}
		if _, ok := tmpl.NodeSelector[""]; ok {
			t.Fatalf("empty key must not be applied; got %#v", tmpl.NodeSelector)
		}
		if got, ok := tmpl.NodeSelector["cloud.google.com/gke-nodepool"]; ok && got == "" {
			t.Fatalf("empty value must not be applied; got %#v", tmpl.NodeSelector)
		}
		return
	}
	t.Fatal("step-gpu template not found")
}

func TestTranspileAppliesTemplateSchedulingDefaults(t *testing.T) {
	p := &Pipeline{
		Name: "target-scheduling",
		Nodes: []Node{{
			ID: "step-a",
			Component: Component{
				Name:  "worker",
				Image: "busybox:latest",
			},
		}},
	}
	wf, err := Transpile(p, &Options{
		Name: "target-scheduling",
		TemplateNodeSelector: map[string]string{
			"workload": "databrew",
		},
		TemplateTolerations: []corev1.Toleration{{
			Key:      "environment",
			Operator: corev1.TolerationOpEqual,
			Value:    "dev",
			Effect:   corev1.TaintEffectNoSchedule,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name != "step-worker" {
			continue
		}
		if got := tmpl.NodeSelector["workload"]; got != "databrew" {
			t.Fatalf("node selector = %q, want databrew", got)
		}
		assertTemplateToleration(t, tmpl, "environment", "dev")
		return
	}
	t.Fatal("step-worker template not found")
}

func TestTranspileEmitsCSISecretProviderClassVolume(t *testing.T) {
	p := &Pipeline{
		Name: "secret-mount",
		Nodes: []Node{{
			ID: "secret-step",
			Component: Component{
				Name:  "secret",
				Image: "busybox:latest",
			},
			VolumeMounts: []VolumeMount{{
				Name:                   "secret-vol",
				MountPath:              "/mnt/secrets",
				ReadOnly:               true,
				CSISecretProviderClass: "db-secret-provider",
			}},
		}},
	}

	wf, err := Transpile(p, &Options{Name: "secret-mount"})
	if err != nil {
		t.Fatal(err)
	}
	var volume *corev1.Volume
	for i := range wf.Spec.Volumes {
		if wf.Spec.Volumes[i].Name == "secret-vol" {
			volume = &wf.Spec.Volumes[i]
			break
		}
	}
	if volume == nil || volume.CSI == nil {
		t.Fatalf("expected CSI volume, got %#v", wf.Spec.Volumes)
	}
	if volume.CSI.Driver != "secrets-store-gke.csi.k8s.io" {
		t.Fatalf("CSI driver = %q", volume.CSI.Driver)
	}
	if volume.CSI.VolumeAttributes["secretProviderClass"] != "db-secret-provider" { // pragma: allowlist secret
		t.Fatalf("CSI attrs = %#v", volume.CSI.VolumeAttributes)
	}
	if volume.CSI.ReadOnly == nil || !*volume.CSI.ReadOnly {
		t.Fatal("expected CSI readOnly true")
	}
	var mounted bool
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name != "step-secret" || tmpl.Container == nil {
			continue
		}
		for _, mount := range tmpl.Container.VolumeMounts {
			if mount.Name == "secret-vol" && mount.MountPath == "/mnt/secrets" && mount.ReadOnly {
				mounted = true
			}
		}
	}
	if !mounted {
		t.Fatal("expected secret volume mount in node template")
	}
}

func TestTranspileEmitsPVCAndEmptyDirRuntimeVolumes(t *testing.T) {
	p := &Pipeline{
		Name: "storage-mount",
		Nodes: []Node{{
			ID: "storage-step",
			Component: Component{
				Name:  "storage",
				Image: "busybox:latest",
			},
			VolumeMounts: []VolumeMount{
				{Name: "model-pvc", MountPath: "/workspace/models", PVCName: "shared-models", ReadOnly: true},
				{Name: "scratch", MountPath: "/workspace/scratch", EmptyDir: true},
			},
		}},
	}

	wf, err := Transpile(p, &Options{Name: "storage-mount"})
	if err != nil {
		t.Fatal(err)
	}
	volumes := map[string]corev1.Volume{}
	for _, volume := range wf.Spec.Volumes {
		volumes[volume.Name] = volume
	}
	if volumes["model-pvc"].PersistentVolumeClaim == nil || volumes["model-pvc"].PersistentVolumeClaim.ClaimName != "shared-models" {
		t.Fatalf("expected shared-models PVC, got %#v", volumes["model-pvc"])
	}
	if !volumes["model-pvc"].PersistentVolumeClaim.ReadOnly {
		t.Fatal("expected PVC readOnly true")
	}
	if volumes["scratch"].EmptyDir == nil {
		t.Fatalf("expected scratch emptyDir, got %#v", volumes["scratch"])
	}
}

func TestTranspileEmitsRuntimeVolumeMountsForScriptNodes(t *testing.T) {
	p := &Pipeline{
		Name: "script-storage-mount",
		Nodes: []Node{{
			ID: "script-step",
			Component: Component{
				Name:    "script",
				Image:   "python:3.12-alpine",
				Mode:    "script",
				Command: []string{"python"},
				Source:  "print('ok')",
			},
			VolumeMounts: []VolumeMount{{
				Name:      "scratch",
				MountPath: "/workspace/scratch",
				EmptyDir:  true,
			}},
		}},
	}

	wf, err := Transpile(p, &Options{Name: "script-storage-mount"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name != "step-script" || tmpl.Script == nil {
			continue
		}
		for _, mount := range tmpl.Script.VolumeMounts {
			if mount.Name == "scratch" && mount.MountPath == "/workspace/scratch" {
				return
			}
		}
		t.Fatalf("expected script volume mount, got %#v", tmpl.Script.VolumeMounts)
	}
	t.Fatal("step-script script template not found")
}

func TestTranspileRejectsConsumedOutputWithoutFileWrite(t *testing.T) {
	p := &Pipeline{
		Name: "missing-output",
		Nodes: []Node{
			{
				ID: "a",
				Component: Component{
					Name:    "a",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo a"}},
				},
				Outputs: []Port{{Name: "output", Type: "string"}},
			},
			{
				ID: "b",
				Component: Component{
					Name:    "b",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c"},
					Args:    []Argument{{Name: "script", Value: "echo b"}},
				},
				Inputs: []Port{{Name: "input", Type: "string"}},
			},
		},
		Edges: []Edge{{Source: "a.output", Target: "b.input"}},
	}

	_, err := Transpile(p, &Options{Name: "missing-output"})
	if err == nil {
		t.Fatal("expected consumed output file error")
	}
	if !strings.Contains(err.Error(), "/tmp/outputs/output") {
		t.Fatalf("error = %q, want output path detail", err.Error())
	}
}

func TestTranspileNormalizesDuplicatedShellArgs(t *testing.T) {
	p := &Pipeline{
		Name: "normalized-shell",
		Nodes: []Node{{
			ID: "n1",
			Component: Component{
				Name:    "n",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c"},
				Args: []Argument{
					{Name: "sh", Value: "sh"},
					{Name: "-c", Value: "-c"},
					{Name: "script", Value: "echo ok"},
				},
			},
		}},
	}

	wf, err := Transpile(p, &Options{Name: "normalized-shell"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "step-n" {
			if got := tmpl.Container.Args; len(got) != 1 || got[0] != "echo ok" {
				t.Fatalf("args = %#v, want single script body", got)
			}
			return
		}
	}
	t.Fatal("step-n1 template not found")
}

func TestTranspileSkipsUnconsumedOutputFileContract(t *testing.T) {
	p := &Pipeline{
		Name: "unconsumed-output",
		Nodes: []Node{{
			ID: "n1",
			Component: Component{
				Name:    "n",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c"},
				Args:    []Argument{{Name: "script", Value: "echo ok"}},
			},
			Outputs: []Port{{Name: "output", Type: "string"}},
		}},
	}

	wf, err := Transpile(p, &Options{Name: "unconsumed-output"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "step-n" {
			if len(tmpl.Outputs.Parameters) != 0 {
				t.Fatalf("outputs = %+v, want none for unconsumed output port", tmpl.Outputs.Parameters)
			}
			if len(tmpl.Container.Args) > 0 && tmpl.Container.Args[len(tmpl.Container.Args)-1] != "echo ok" {
				t.Fatalf("container args = %v, want command unchanged", tmpl.Container.Args)
			}
			return
		}
	}
	t.Fatal("step-n1 template not found")
}

func TestTranspileDeclaresOutputWhenCommandWritesOutputFile(t *testing.T) {
	p := &Pipeline{
		Name: "file-output",
		Nodes: []Node{{
			ID: "n1",
			Component: Component{
				Name:    "n",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c"},
				Args:    []Argument{{Name: "script", Value: "echo ok > /tmp/outputs/output"}},
			},
			Outputs: []Port{{Name: "output", Type: "string"}},
		}},
	}

	wf, err := Transpile(p, &Options{Name: "file-output"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "step-n" {
			if len(tmpl.Outputs.Parameters) != 1 || tmpl.Outputs.Parameters[0].Name != "output" {
				t.Fatalf("outputs = %+v, want output parameter", tmpl.Outputs.Parameters)
			}
			if got := tmpl.Container.Args[len(tmpl.Container.Args)-1]; got != "mkdir -p /tmp/outputs && echo ok > /tmp/outputs/output" {
				t.Fatalf("script arg = %q", got)
			}
			return
		}
	}
	t.Fatal("step-n1 template not found")
}

func TestTranspileDefaultTTL(t *testing.T) {
	p := &Pipeline{
		Name: "ttl",
		Nodes: []Node{{
			ID:        "n1",
			Component: Component{Name: "n", Image: "busybox:latest"},
		}},
	}
	wf, err := Transpile(p, &Options{Name: "ttl-test"})
	if err != nil {
		t.Fatal(err)
	}
	if wf.Spec.TTLStrategy == nil || wf.Spec.TTLStrategy.SecondsAfterCompletion == nil {
		t.Fatal("expected TTL strategy")
	}
	if *wf.Spec.TTLStrategy.SecondsAfterCompletion != DefaultTTLSecondsAfterCompletion {
		t.Fatalf("ttl = %d, want %d", *wf.Spec.TTLStrategy.SecondsAfterCompletion, DefaultTTLSecondsAfterCompletion)
	}
}

func TestTranspilePodGC(t *testing.T) {
	p := &Pipeline{
		Name: "podgc",
		Nodes: []Node{{
			ID:        "n1",
			Component: Component{Name: "n", Image: "busybox:latest"},
		}},
	}
	wf, err := Transpile(p, &Options{Name: "podgc-test"})
	if err != nil {
		t.Fatal(err)
	}
	if wf.Spec.PodGC == nil {
		t.Fatal("expected PodGC strategy")
	}
	if wf.Spec.PodGC.Strategy != wfv1.PodGCOnWorkflowSuccess {
		t.Fatalf("podGC strategy = %q, want %q", wf.Spec.PodGC.Strategy, wfv1.PodGCOnWorkflowSuccess)
	}
	// Post-CYB-3667 tightening: keep step pods around for 24h after a workflow
	// succeeds so the GCP-console deep link, kubectl logs / kubectl describe, and
	// the frontend "查看 Pod" button keep working through a normal after-hours
	// operator window. Bare OnWorkflowSuccess (no delay) tore pods down the
	// instant a workflow finished and broke every deep link (see #494).
	if got, want := wf.Spec.PodGC.DeleteDelayDuration, "24h"; got != want {
		t.Fatalf("podGC deleteDelayDuration = %q, want %q", got, want)
	}
	if _, err := wf.Spec.PodGC.GetDeleteDelayDuration(); err != nil {
		// Guardrail: string form must parse into a time.Duration so the argo
		// controller accepts it. Anything the SDK's own parser rejects here
		// would silently drop back to immediate GC in the cluster.
		t.Fatalf("podGC deleteDelayDuration %q does not parse as time.Duration: %v",
			wf.Spec.PodGC.DeleteDelayDuration, err)
	}
}

func TestTranspileRetryStrategy(t *testing.T) {
	p := &Pipeline{
		Name: "retry-test",
		Nodes: []Node{{
			ID:        "n1",
			Component: Component{Name: "n", Image: "busybox:latest"},
		}},
	}
	opts := &Options{
		Name:          "retry-test",
		RetryStrategy: &RetryStrategy{Limit: 3},
	}
	wf, err := Transpile(p, opts)
	if err != nil {
		t.Fatal(err)
	}

	// Find the node template (not the DAG template)
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "step-n" {
			if tmpl.RetryStrategy == nil {
				t.Fatal("expected retry strategy")
			}
			if tmpl.RetryStrategy.Limit.IntValue() != 3 {
				t.Fatalf("retry limit = %d, want 3", tmpl.RetryStrategy.Limit.IntValue())
			}
			return
		}
	}
	t.Fatal("step-n1 template not found")
}

func TestTranspileActiveDeadlineSeconds(t *testing.T) {
	p := &Pipeline{
		Name: "timeout-test",
		Nodes: []Node{{
			ID:        "n1",
			Component: Component{Name: "n", Image: "busybox:latest"},
		}},
	}
	opts := &Options{
		Name:                  "timeout-test",
		ActiveDeadlineSeconds: 600,
	}
	wf, err := Transpile(p, opts)
	if err != nil {
		t.Fatal(err)
	}

	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "step-n" {
			if tmpl.ActiveDeadlineSeconds == nil {
				t.Fatal("expected activeDeadlineSeconds")
			}
			if tmpl.ActiveDeadlineSeconds.IntValue() != 600 {
				t.Fatalf("activeDeadlineSeconds = %d, want 600", tmpl.ActiveDeadlineSeconds.IntValue())
			}
			return
		}
	}
	t.Fatal("step-n1 template not found")
}

func TestTranspileRetryAndTimeout(t *testing.T) {
	p := &Pipeline{
		Name: "combined",
		Nodes: []Node{{
			ID:        "n1",
			Component: Component{Name: "n", Image: "busybox:latest"},
		}},
	}
	opts := &Options{
		Name:                  "combined",
		RetryStrategy:         &RetryStrategy{Limit: 2},
		ActiveDeadlineSeconds: 300,
	}
	wf, err := Transpile(p, opts)
	if err != nil {
		t.Fatal(err)
	}

	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "step-n" {
			if tmpl.RetryStrategy == nil || tmpl.RetryStrategy.Limit.IntValue() != 2 {
				t.Fatal("expected retry limit 2")
			}
			if tmpl.ActiveDeadlineSeconds == nil || tmpl.ActiveDeadlineSeconds.IntValue() != 300 {
				t.Fatal("expected timeout 300s")
			}
			return
		}
	}
	t.Fatal("step-n1 template not found")
}

func TestTranspileWorkflowParams(t *testing.T) {
	p := &Pipeline{
		Name: "wf-params",
		Nodes: []Node{{
			ID:        "n1",
			Component: Component{Name: "n", Image: "busybox:latest"},
		}},
	}
	opts := &Options{
		Name: "wf-params",
		WorkflowParams: []Param{
			{Name: "asset_ids", Value: "[\"V4ftKjqX\",\"kK9qIAqG\"]"},
			{Name: "threshold", Value: "0.5"},
		},
	}
	wf, err := Transpile(p, opts)
	if err != nil {
		t.Fatal(err)
	}

	if len(wf.Spec.Arguments.Parameters) != 2 {
		t.Fatalf("got %d params, want 2", len(wf.Spec.Arguments.Parameters))
	}

	var names []string
	for _, p := range wf.Spec.Arguments.Parameters {
		names = append(names, p.Name)
		if p.Value == nil || p.Value.String() == "" {
			t.Fatalf("param %q has nil or empty value", p.Name)
		}
	}

	if names[0] != "asset_ids" || names[1] != "threshold" {
		t.Fatalf("params = %v, want [asset_ids threshold]", names)
	}
}

func TestTranspileParallelism(t *testing.T) {
	p := &Pipeline{
		Name:        "parallel-test",
		Parallelism: 3,
		Nodes: []Node{{
			ID:        "n1",
			Component: Component{Name: "n", Image: "busybox:latest"},
		}},
	}
	wf, err := Transpile(p, &Options{Name: "parallel-test"})
	if err != nil {
		t.Fatal(err)
	}
	if wf.Spec.Parallelism == nil {
		t.Fatal("expected Parallelism to be set")
	}
	if *wf.Spec.Parallelism != 3 {
		t.Fatalf("Parallelism = %d, want 3", *wf.Spec.Parallelism)
	}
}

func TestTranspileUsesCanonicalStepTemplateName(t *testing.T) {
	p := &Pipeline{
		Name: "wf",
		Nodes: []Node{{
			ID:        "step-1",
			Component: Component{Name: "sleep", Image: "alpine:3.20", Command: []string{"sh", "-c"}, Args: []Argument{{Name: "cmd", Value: "sleep 1"}}},
		}},
	}
	wf, err := Transpile(p, &Options{Name: "wf-test", Namespace: "default"})
	if err != nil {
		t.Fatal(err)
	}
	var dag *wfv1.Template
	for i := range wf.Spec.Templates {
		if wf.Spec.Templates[i].Name == "dag" {
			dag = &wf.Spec.Templates[i]
			break
		}
	}
	if dag == nil || dag.DAG == nil || len(dag.DAG.Tasks) != 1 {
		t.Fatalf("expected one DAG task, got %#v", dag)
	}
	if dag.DAG.Tasks[0].Name != "step-sleep" {
		t.Fatalf("task name = %q, want step-sleep", dag.DAG.Tasks[0].Name)
	}
	if dag.DAG.Tasks[0].Template != "step-sleep" {
		t.Fatalf("task template = %q, want step-sleep", dag.DAG.Tasks[0].Template)
	}
}

// helper: a minimal single-node pipeline for hook tests.
func singleNodePipeline() *Pipeline {
	return &Pipeline{
		Name: "hooktest",
		Nodes: []Node{{
			ID: "a",
			Component: Component{
				Name:    "a",
				Image:   "busybox:latest",
				Command: []string{"sh", "-c"},
				Args:    []Argument{{Name: "script", Value: "echo hi"}},
			},
		}},
	}
}

func findTemplate(wf *wfv1.Workflow, name string) *wfv1.Template {
	for i := range wf.Spec.Templates {
		if wf.Spec.Templates[i].Name == name {
			return &wf.Spec.Templates[i]
		}
	}
	return nil
}

func TestTranspileInjectsExitHookWhenURLSet(t *testing.T) {
	wf, err := Transpile(singleNodePipeline(), &Options{
		Name:                    "wf-hook",
		Namespace:               "cyber-databrew-dev",
		ExitHookURL:             "https://backend.example/api/v1/pipeline-runs/webhook",
		ExitHookTokenSecretName: "databrew-run-webhook-token",
		ExitHookTokenSecretKey:  "token",
	})
	if err != nil {
		t.Fatal(err)
	}
	hook, ok := wf.Spec.Hooks[wfv1.ExitLifecycleEvent]
	if !ok {
		t.Fatalf("expected exit lifecycle hook to be set")
	}
	if hook.Template != ExitNotifyTemplateName {
		t.Fatalf("hook template = %q, want %q", hook.Template, ExitNotifyTemplateName)
	}
	// The exit handler must be a plain container (NOT an Argo http template,
	// whose agent pod cannot start on this cluster), running curl.
	tmpl := findTemplate(wf, ExitNotifyTemplateName)
	if tmpl == nil || tmpl.Container == nil {
		t.Fatalf("expected container notify template %q", ExitNotifyTemplateName)
	}
	if tmpl.HTTP != nil {
		t.Fatalf("notify template must not use the http template (agent pod unavailable)")
	}
	if tmpl.Container.Image == "" {
		t.Fatalf("notify container must set an image")
	}
	// Minimal, explicit resource footprint (curl once-off).
	if tmpl.Container.Resources.Requests.Cpu().IsZero() || tmpl.Container.Resources.Requests.Memory().IsZero() {
		t.Fatalf("notify container must set minimal cpu/memory requests, got %+v", tmpl.Container.Resources)
	}
	if tmpl.Container.Resources.Limits.Cpu().IsZero() || tmpl.Container.Resources.Limits.Memory().IsZero() {
		t.Fatalf("notify container must set cpu/memory limits, got %+v", tmpl.Container.Resources)
	}
	joined := strings.Join(tmpl.Container.Args, " ")
	if !strings.Contains(joined, "curl") || !strings.Contains(joined, "$DATABREW_WEBHOOK_URL") {
		t.Fatalf("notify container must curl the webhook url: %q", joined)
	}
	// URL as env value; token as env valueFrom.secretKeyRef (never literal).
	var urlEnv, tokEnv *corev1.EnvVar
	for i := range tmpl.Container.Env {
		switch tmpl.Container.Env[i].Name {
		case "DATABREW_WEBHOOK_URL":
			urlEnv = &tmpl.Container.Env[i]
		case "DATABREW_WEBHOOK_TOKEN":
			tokEnv = &tmpl.Container.Env[i]
		}
	}
	if urlEnv == nil || urlEnv.Value != "https://backend.example/api/v1/pipeline-runs/webhook" {
		t.Fatalf("unexpected webhook url env: %+v", urlEnv)
	}
	if tokEnv == nil {
		t.Fatalf("expected DATABREW_WEBHOOK_TOKEN env")
	}
	if tokEnv.Value != "" {
		t.Fatalf("token env must not carry a literal value")
	}
	if tokEnv.ValueFrom == nil || tokEnv.ValueFrom.SecretKeyRef == nil {
		t.Fatalf("token env must use valueFrom.secretKeyRef")
	}
	if tokEnv.ValueFrom.SecretKeyRef.Name != "databrew-run-webhook-token" || tokEnv.ValueFrom.SecretKeyRef.Key != "token" { // pragma: allowlist secret
		t.Fatalf("unexpected secretKeyRef: %+v", tokEnv.ValueFrom.SecretKeyRef)
	}
}

// TestTranspileSetsPodPriorityClassName verifies pool.2 (CYB-3486) injects
// the target's PriorityClass onto the workflow spec so every pod inherits it.
func TestTranspileSetsPodPriorityClassName(t *testing.T) {
	wf, err := Transpile(singleNodePipeline(), &Options{
		Name:                 "wf-prio",
		Namespace:            "default",
		PodPriorityClassName: "cyber-databrew-batch",
	})
	if err != nil {
		t.Fatal(err)
	}
	if wf.Spec.PodPriorityClassName != "cyber-databrew-batch" {
		t.Errorf("PodPriorityClassName: want %q, got %q", "cyber-databrew-batch", wf.Spec.PodPriorityClassName)
	}
}

// TestTranspileOmitsPodPriorityClassNameWhenEmpty guards the byte-identical
// backward-compat: empty option leaves the field unset so K8s default
// scheduling behavior is preserved for pre-pool.2 rows.
func TestTranspileOmitsPodPriorityClassNameWhenEmpty(t *testing.T) {
	wf, err := Transpile(singleNodePipeline(), &Options{Name: "wf-noprio", Namespace: "default"})
	if err != nil {
		t.Fatal(err)
	}
	if wf.Spec.PodPriorityClassName != "" {
		t.Errorf("empty option must leave PodPriorityClassName empty, got %q", wf.Spec.PodPriorityClassName)
	}
}

// TestTranspileSetsSchedulerName verifies pool (CYB-3486) injects the pool's
// scheduler onto the workflow spec. Data-driven: transpiler sets whatever
// string it's given, no hardcoded scheduler.
func TestTranspileSetsSchedulerName(t *testing.T) {
	wf, err := Transpile(singleNodePipeline(), &Options{
		Name:          "wf-sched",
		Namespace:     "default",
		SchedulerName: "koord-scheduler",
	})
	if err != nil {
		t.Fatal(err)
	}
	if wf.Spec.SchedulerName != "koord-scheduler" {
		t.Errorf("SchedulerName: want koord-scheduler, got %q", wf.Spec.SchedulerName)
	}
}

// TestTranspileOmitsSchedulerNameWhenEmpty: scheduler-agnostic pool leaves the
// field unset so the cluster default scheduler runs the pods.
func TestTranspileOmitsSchedulerNameWhenEmpty(t *testing.T) {
	wf, err := Transpile(singleNodePipeline(), &Options{Name: "wf-nosched", Namespace: "default"})
	if err != nil {
		t.Fatal(err)
	}
	if wf.Spec.SchedulerName != "" {
		t.Errorf("empty option must leave SchedulerName unset, got %q", wf.Spec.SchedulerName)
	}
}

// TestTranspileSetsPodAnnotations verifies pool pod annotations reach
// Spec.PodMetadata.Annotations.
func TestTranspileSetsPodAnnotations(t *testing.T) {
	wf, err := Transpile(singleNodePipeline(), &Options{
		Name:           "wf-anno",
		Namespace:      "default",
		PodAnnotations: map[string]string{"scheduling.koordinator.sh/tier": "batch"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if wf.Spec.PodMetadata == nil || wf.Spec.PodMetadata.Annotations["scheduling.koordinator.sh/tier"] != "batch" {
		t.Errorf("pod annotation not injected: %+v", wf.Spec.PodMetadata)
	}
}

func TestTranspileNoExitHookWhenURLEmpty(t *testing.T) {
	wf, err := Transpile(singleNodePipeline(), &Options{Name: "wf-nohook", Namespace: "default"})
	if err != nil {
		t.Fatal(err)
	}
	if len(wf.Spec.Hooks) != 0 {
		t.Fatalf("expected no hooks, got %v", wf.Spec.Hooks)
	}
	if findTemplate(wf, ExitNotifyTemplateName) != nil {
		t.Fatalf("notify template must not be present when hook disabled")
	}
}
