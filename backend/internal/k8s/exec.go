package k8s

import (
	"context"
	"fmt"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type ExecClient interface {
	ExecPod(ctx context.Context, req PodExecRequest, stdout, stderr io.Writer) error
}

type PodExecRequest struct {
	Namespace     string
	PodName       string
	ContainerName string
	Command       []string
	TTY           bool
}

type execClient struct {
	clientset kubernetes.Interface
	config    *rest.Config
}

func NewExecClient(kubeconfigPath string) (ExecClient, error) {
	config, err := buildConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("%w: create clientset: %v", ErrUnavailable, err)
	}
	return NewExecClientWithConfig(clientset, config), nil
}

func NewExecClientWithConfig(clientset kubernetes.Interface, config *rest.Config) ExecClient {
	return &execClient{clientset: clientset, config: rest.CopyConfig(config)}
}

func (c *execClient) ExecPod(ctx context.Context, req PodExecRequest, stdout, stderr io.Writer) error {
	if c == nil || c.clientset == nil || c.config == nil {
		return fmt.Errorf("%w: k8s exec client is not configured", ErrUnavailable)
	}
	command := sanitizeExecCommand(req.Command)
	if len(command) == 0 {
		return fmt.Errorf("pod exec command is required")
	}
	request := c.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(req.PodName).
		Namespace(req.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: req.ContainerName,
			Command:   command,
			Stdout:    true,
			Stderr:    true,
			TTY:       req.TTY,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(c.config, "POST", request.URL())
	if err != nil {
		return fmt.Errorf("create pod exec stream: %w", err)
	}
	if err := executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: stdout,
		Stderr: stderr,
		Tty:    req.TTY,
	}); err != nil {
		return fmt.Errorf("stream pod exec: %w", err)
	}
	return nil
}

func sanitizeExecCommand(command []string) []string {
	out := make([]string, 0, len(command))
	for _, part := range command {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
