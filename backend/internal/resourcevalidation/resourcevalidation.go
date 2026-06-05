package resourcevalidation

import (
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

var memoryOrDiskUnitPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?(?:Ki|Mi|Gi|Ti|Pi|Ei|K|M|G|T|P|E)$`)
var gpuPattern = regexp.MustCompile(`^[0-9]+$`)

// ValidateResourceStrings applies DataBrew resource semantics on top of
// Kubernetes quantity parsing. Kubernetes accepts bare memory values as bytes;
// DataBrew requires explicit units so users do not accidentally request "1B".
func ValidateResourceStrings(context string, cpu, memory, disk, gpu string) []string {
	var problems []string
	problems = append(problems, validateKubernetesQuantity(context, "CPU", cpu)...)
	problems = append(problems, validateMemoryOrDisk(context, "内存", memory, "512Mi、1Gi")...)
	problems = append(problems, validateMemoryOrDisk(context, "磁盘", disk, "1Gi、20Gi")...)
	if strings.TrimSpace(gpu) != "" && !gpuPattern.MatchString(strings.TrimSpace(gpu)) {
		problems = append(problems, fmt.Sprintf("%s GPU must be a non-negative integer, for example 0, 1, or 2", context))
	}
	if strings.TrimSpace(gpu) != "" {
		problems = append(problems, validateKubernetesQuantity(context, "GPU", gpu)...)
	}
	return problems
}

func validateKubernetesQuantity(context, label, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	q, err := resource.ParseQuantity(value)
	if err != nil {
		return []string{fmt.Sprintf("%s %s quantity %q is invalid: %v", context, label, value, err)}
	}
	if label != "GPU" && q.Sign() <= 0 {
		return []string{fmt.Sprintf("%s %s quantity %q must be positive", context, label, value)}
	}
	return nil
}

func validateMemoryOrDisk(context, label, value, examples string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	q, err := resource.ParseQuantity(value)
	if err != nil {
		return []string{fmt.Sprintf("%s %s quantity %q is invalid: %v", context, label, value, err)}
	}
	if q.Sign() <= 0 {
		return []string{fmt.Sprintf("%s %s quantity %q must be positive", context, label, value)}
	}
	if !memoryOrDiskUnitPattern.MatchString(value) {
		return []string{fmt.Sprintf("%s %s must include a unit, for example %s", context, label, examples)}
	}
	return nil
}
