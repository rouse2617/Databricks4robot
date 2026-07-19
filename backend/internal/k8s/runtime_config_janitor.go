package k8s

import (
	"context"
	"log/slog"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Runtime-config janitor (CYB-3680): content-addressed ConfigMaps are shared
// and owner-less, so Kubernetes never garbage-collects them. This sweep
// deletes only CMs whose sliding-reference annotation aged past the TTL —
// a CM any active batch still ensures gets its timestamp refreshed and is
// therefore never touched. Deleting a CM is never data loss: the DB blob is
// the source of truth and the next ensure recreates it.
//
// Scope note (v1.5 §06): this janitor sweeps ONE cluster (the clientset it is
// given — wired for the default cluster). Per-cluster sweeps ride the
// per-cluster channel loops of CYB-3678/3681. Legacy per-run CMs are excluded
// by the label selector and keep their owner-cascade lifecycle.

// SweepRuntimeConfigs deletes expired content-addressed runtime-config CMs
// cluster-wide and returns how many were deleted. Unparsable or missing
// last-referenced annotations are treated as fresh (safety: never delete on
// ambiguity).
func SweepRuntimeConfigs(ctx context.Context, clientset kubernetes.Interface, ttl time.Duration) (int, error) {
	list, err := clientset.CoreV1().ConfigMaps(metav1.NamespaceAll).List(ctx, metav1.ListOptions{
		LabelSelector: LabelContentAddressed + "=true",
	})
	if err != nil {
		return 0, err
	}
	deleted := 0
	now := time.Now().UTC()
	for i := range list.Items {
		cm := &list.Items[i]
		raw, ok := cm.Annotations[AnnotationLastReferenced]
		if !ok {
			continue // ambiguity → keep
		}
		ts, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			continue // ambiguity → keep
		}
		if now.Sub(ts) <= ttl {
			continue
		}
		if err := clientset.CoreV1().ConfigMaps(cm.Namespace).Delete(ctx, cm.Name, metav1.DeleteOptions{}); err != nil {
			slog.Warn("runtime config janitor delete failed", "namespace", cm.Namespace, "name", cm.Name, "err", err)
			continue
		}
		deleted++
		slog.Info("runtime config janitor reclaimed expired ConfigMap",
			"namespace", cm.Namespace, "name", cm.Name, "lastReferenced", raw)
	}
	return deleted, nil
}

// StartRuntimeConfigJanitor runs SweepRuntimeConfigs on an interval until ctx
// is cancelled. One eager sweep on boot.
func StartRuntimeConfigJanitor(ctx context.Context, clientset kubernetes.Interface, ttl, interval time.Duration) {
	go func() {
		sweep := func() {
			if n, err := SweepRuntimeConfigs(ctx, clientset, ttl); err != nil {
				slog.Warn("runtime config janitor sweep failed", "err", err)
			} else if n > 0 {
				slog.Info("runtime config janitor sweep done", "deleted", n)
			}
		}
		sweep()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sweep()
			}
		}
	}()
}
