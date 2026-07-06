package transpiler

import "strings"

// stepSlugMaxLen bounds the readable slug so pod names
// (<wfName>-<templateName>-<hash>) stay well within the K8s 253-char limit.
const stepSlugMaxLen = 30

// stepSlug renders a component name into a readable, RFC1123-safe slug:
// lowercase, [a-z0-9] with other runs collapsed to '-', trimmed, truncated.
// (Kept in-package to avoid depending on the handlers layer.)
func stepSlug(componentName string) string {
	value := strings.ToLower(strings.TrimSpace(componentName))
	var out strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(out.String(), "-")
	if len(slug) > stepSlugMaxLen {
		slug = strings.Trim(slug[:stepSlugMaxLen], "-")
	}
	return slug
}

// nodeUUIDHex returns the node id's hex payload (strip the "node-" prefix and
// separators), used to disambiguate steps that share a component name.
func nodeUUIDHex(nodeID string) string {
	s := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(nodeID)), "node-")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")
	return s
}

// nodeUUID8 returns the first 8 hex chars of the node id (or the whole payload
// if shorter).
func nodeUUID8(nodeID string) string {
	hex := nodeUUIDHex(nodeID)
	if len(hex) > 8 {
		return hex[:8]
	}
	return hex
}

// flattenNodes returns all nodes in the pipeline, recursing into sub-graphs, so
// naming and collision checks operate over the full flat set of Argo templates.
func flattenNodes(nodes []Node) []Node {
	var flat []Node
	var walk func(ns []Node)
	walk = func(ns []Node) {
		for _, n := range ns {
			flat = append(flat, n)
			if len(n.SubNodes) > 0 {
				walk(n.SubNodes)
			}
		}
	}
	walk(nodes)
	return flat
}

// buildStepTemplateNames maps every node id (top-level + nested sub-nodes) to
// its Argo template name — the single source of truth for pod-name readability.
//
//	step-<component-slug>              when the slug is unique in the workflow
//	step-<component-slug>-<uuid8>      when >1 node shares the slug
//	step-<nodeID>                      when the component name is empty (legacy)
//
// Template names must be unique across the whole workflow (Argo rejects
// duplicates), so a final pass extends any residual collision with the full
// node hex. Deterministic given node order.
func buildStepTemplateNames(nodes []Node) map[string]string {
	flat := flattenNodes(nodes)

	slugs := make(map[string]string, len(flat))
	slugCount := make(map[string]int)
	for _, n := range flat {
		s := stepSlug(n.Component.Name)
		slugs[n.ID] = s
		if s != "" {
			slugCount[s]++
		}
	}

	names := make(map[string]string, len(flat))
	used := make(map[string]string, len(flat)) // name -> first node id
	for _, n := range flat {
		s := slugs[n.ID]
		var name string
		switch {
		case s == "":
			name = templateName(n.ID) // legacy fallback for unnamed components
		case slugCount[s] > 1:
			name = "step-" + s + "-" + nodeUUID8(n.ID)
		default:
			name = "step-" + s
		}
		// Guarantee global uniqueness (uuid8 collision or fallback overlap).
		if prior, ok := used[name]; ok && prior != n.ID && s != "" {
			name = "step-" + s + "-" + nodeUUIDHex(n.ID)
		}
		names[n.ID] = name
		if _, ok := used[name]; !ok {
			used[name] = n.ID
		}
	}
	return names
}

// StepCandidateKeys returns every template name a definition node could have
// been stored as, so run views relate stored pipeline_node_id values (old or
// new format) back to the definition without a data migration:
//
//	legacy  step-<nodeID>
//	new     step-<slug>, step-<slug>-<uuid8>, step-<slug>-<fullHex>
//
// It is intentionally dup-unaware: offering all forms is safe because each form
// is derived from this node's own id, and the transpiler only ever emitted one
// of them for this node.
func StepCandidateKeys(componentName, nodeID string) []string {
	legacy := templateName(nodeID)
	slug := stepSlug(componentName)
	if slug == "" {
		return []string{legacy}
	}
	out := []string{
		legacy,
		"step-" + slug,
		"step-" + slug + "-" + nodeUUID8(nodeID),
		"step-" + slug + "-" + nodeUUIDHex(nodeID),
	}
	seen := make(map[string]bool, len(out))
	deduped := out[:0]
	for _, k := range out {
		if seen[k] {
			continue
		}
		seen[k] = true
		deduped = append(deduped, k)
	}
	return deduped
}
