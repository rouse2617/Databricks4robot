package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cyber-databrew/databrew-pipeline/transpiler"
	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: dbtp <pipeline.json> [flags]\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fmt.Fprintf(os.Stderr, "  --yaml         Output Argo Workflow as YAML (default: JSON)\n")
		fmt.Fprintf(os.Stderr, "  --namespace    K8s namespace (default: default)\n")
		fmt.Fprintf(os.Stderr, "  --name         Workflow name (default: pipeline name)\n")
		fmt.Fprintf(os.Stderr, "  --sa           K8s service account\n")
		fmt.Fprintf(os.Stderr, "  --pull-secret  Image pull secret (repeatable)\n")
		fmt.Fprintf(os.Stderr, "  --ttl          TTL seconds after completion (default: 3600)\n")
		os.Exit(1)
	}

	pipeFile := os.Args[1]

	// Parse flags
	opts := parseFlags(os.Args[2:])

	// Read pipeline JSON
	data, err := os.ReadFile(pipeFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", pipeFile, err)
		os.Exit(1)
	}

	var pipe transpiler.Pipeline
	if err := json.Unmarshal(data, &pipe); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing pipeline JSON: %v\n", err)
		os.Exit(1)
	}

	// Transpile
	wf, err := transpiler.Transpile(&pipe, opts.opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Transpile error: %v\n", err)
		os.Exit(1)
	}

	// Output
	if opts.yamlOut {
		out, err := yaml.Marshal(wf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "YAML marshal error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(string(out))
	} else {
		out, err := json.MarshalIndent(wf, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "JSON marshal error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(string(out))
	}
}

type cliFlags struct {
	yamlOut bool
	opts    *transpiler.Options
}

func parseFlags(args []string) *cliFlags {
	f := &cliFlags{
		opts: &transpiler.Options{
			TTLSecondsAfter: 3600,
		},
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--yaml":
			f.yamlOut = true
		case "--namespace":
			i++
			if i < len(args) {
				f.opts.Namespace = args[i]
			}
		case "--name":
			i++
			if i < len(args) {
				f.opts.Name = args[i]
			}
		case "--sa":
			i++
			if i < len(args) {
				f.opts.ServiceAccount = args[i]
			}
		case "--pull-secret":
			i++
			if i < len(args) {
				f.opts.ImagePullSecrets = append(f.opts.ImagePullSecrets, args[i])
			}
		case "--ttl":
			i++
			if i < len(args) {
				fmt.Sscanf(args[i], "%d", &f.opts.TTLSecondsAfter)
			}
		}
	}
	return f
}
