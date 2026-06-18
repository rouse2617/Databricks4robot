package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/transpiler"
)

const (
	runtimeSecretKindSecretProviderClass = "secretProviderClass" // pragma: allowlist secret
	runtimeStorageKindPVC                = "pvc"
	runtimeStorageKindEmptyDir           = "emptyDir"
	defaultRuntimeStorageMountPath       = "/workspace/scratch"
)

// RuntimeMountCatalog is the product-facing catalog of mountable runtime
// resources. Users bind catalog IDs to nodes; the backend owns Kubernetes
// SecretProviderClass/PVC details.
type RuntimeMountCatalog struct {
	Secrets []RuntimeSecretMountResource  `json:"secrets"`
	Storage []RuntimeStorageMountResource `json:"storage"`
}

type RuntimeSecretMountResource struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Description         string   `json:"description,omitempty"`
	Kind                string   `json:"kind"`
	SecretProviderClass string   `json:"secretProviderClass"`
	DefaultMountPath    string   `json:"defaultMountPath"`
	ReadOnly            bool     `json:"readOnly"`
	TargetIDs           []string `json:"targetIds,omitempty"`
}

type RuntimeStorageMountResource struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description,omitempty"`
	Kind             string   `json:"kind"`
	PVCName          string   `json:"pvcName,omitempty"`
	DefaultMountPath string   `json:"defaultMountPath"`
	ReadOnly         bool     `json:"readOnly"`
	AllowWrite       bool     `json:"allowWrite"`
	TargetIDs        []string `json:"targetIds,omitempty"`
}

func (uc *Usecase) SetRuntimeMountCatalog(catalog RuntimeMountCatalog) {
	uc.runtimeMountCatalog = catalog.normalized()
}

func (uc *Usecase) ListRuntimeMounts(context.Context) (RuntimeMountCatalog, error) {
	return uc.runtimeMountCatalogOrDefault()
}

func (uc *Usecase) runtimeMountCatalogOrDefault() (RuntimeMountCatalog, error) {
	if len(uc.runtimeMountCatalog.Secrets) > 0 || len(uc.runtimeMountCatalog.Storage) > 0 {
		return uc.runtimeMountCatalog.normalized(), nil
	}
	return defaultRuntimeMountCatalog()
}

func defaultRuntimeMountCatalog() (RuntimeMountCatalog, error) {
	catalog := RuntimeMountCatalog{
		Storage: []RuntimeStorageMountResource{{
			ID:               "scratch-emptydir",
			Name:             "Scratch workspace",
			Description:      "Per-pod emptyDir for intermediate files and temporary cache.",
			Kind:             runtimeStorageKindEmptyDir,
			DefaultMountPath: defaultRuntimeStorageMountPath,
			ReadOnly:         false,
			AllowWrite:       true,
			TargetIDs:        runtimeMountTargetIDsForEnv("PIPELINE_RUNTIME_STORAGE_EMPTYDIR"),
		}},
	}
	catalog.addSecretFromEnv("PIPELINE_RUNTIME_SECRET", "pipeline-secret", "Runtime secret", "/mnt/secrets")
	if err := catalog.mergeRuntimeMountCatalogJSONEnv("PIPELINE_RUNTIME_MOUNT_CATALOG_JSON"); err != nil {
		return RuntimeMountCatalog{}, err
	}
	if err := catalog.mergeRuntimeSecretResourcesJSONEnv("PIPELINE_RUNTIME_SECRET_RESOURCES_JSON"); err != nil {
		return RuntimeMountCatalog{}, err
	}
	if err := catalog.mergeRuntimeStorageResourcesJSONEnv("PIPELINE_RUNTIME_STORAGE_RESOURCES_JSON"); err != nil {
		return RuntimeMountCatalog{}, err
	}
	if pvcName := strings.TrimSpace(os.Getenv("PIPELINE_RUNTIME_STORAGE_PVC_NAME")); pvcName != "" {
		allowWrite := parseBoolEnv("PIPELINE_RUNTIME_STORAGE_PVC_ALLOW_WRITE")
		catalog.addStorageResource(RuntimeStorageMountResource{
			ID:               getenvDefault("PIPELINE_RUNTIME_STORAGE_PVC_ID", "shared-pvc"),
			Name:             getenvDefault("PIPELINE_RUNTIME_STORAGE_PVC_NAME_LABEL", "Shared PVC"),
			Description:      "Platform-provisioned existing PVC for shared data or model directories.",
			Kind:             runtimeStorageKindPVC,
			PVCName:          pvcName,
			DefaultMountPath: getenvDefault("PIPELINE_RUNTIME_STORAGE_PVC_MOUNT_PATH", "/workspace/shared"),
			ReadOnly:         !allowWrite,
			AllowWrite:       allowWrite,
			TargetIDs:        runtimeMountTargetIDsForEnv("PIPELINE_RUNTIME_STORAGE_PVC"),
		})
	}
	return catalog.normalized(), nil
}

func (catalog *RuntimeMountCatalog) addSecretFromEnv(envName, defaultID, defaultName, defaultMountPath string) {
	envValue := strings.TrimSpace(os.Getenv(envName))
	prefixValue := strings.TrimSpace(os.Getenv(envName + "_PROVIDER_CLASS"))
	secretProviderClass := envValue // pragma: allowlist secret
	if prefixValue != "" {
		secretProviderClass = prefixValue // pragma: allowlist secret
	}
	if secretProviderClass == "" {
		return
	}
	id := getenvDefault(envName+"_ID", defaultID)
	name := getenvDefault(envName+"_NAME", defaultName)
	mountPath := getenvDefault(envName+"_MOUNT_PATH", defaultMountPath)
	for _, item := range catalog.Secrets {
		if item.ID == id || item.SecretProviderClass == secretProviderClass { // pragma: allowlist secret
			return
		}
	}
	catalog.addSecretResource(RuntimeSecretMountResource{
		ID:                  id,
		Name:                name,
		Description:         "Platform-provisioned SecretProviderClass mounted read-only into selected nodes.",
		Kind:                runtimeSecretKindSecretProviderClass,
		SecretProviderClass: secretProviderClass,
		DefaultMountPath:    mountPath,
		ReadOnly:            true,
		TargetIDs:           runtimeMountTargetIDsForEnv(envName),
	})
}

func (catalog *RuntimeMountCatalog) mergeRuntimeMountCatalogJSONEnv(envName string) error {
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		return nil
	}
	var parsed RuntimeMountCatalog
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return fmt.Errorf("invalid %s runtime mount catalog JSON: %w", envName, err)
	}
	if err := validateRuntimeMountCatalogResources(envName, parsed.Secrets, parsed.Storage); err != nil {
		return err
	}
	for _, item := range parsed.Secrets {
		catalog.addSecretResource(item)
	}
	for _, item := range parsed.Storage {
		catalog.addStorageResource(item)
	}
	return nil
}

func (catalog *RuntimeMountCatalog) mergeRuntimeSecretResourcesJSONEnv(envName string) error {
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		return nil
	}
	var parsed []RuntimeSecretMountResource
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return fmt.Errorf("invalid %s runtime secret resources JSON: %w", envName, err)
	}
	if err := validateRuntimeMountCatalogResources(envName, parsed, nil); err != nil {
		return err
	}
	for _, item := range parsed {
		catalog.addSecretResource(item)
	}
	return nil
}

func (catalog *RuntimeMountCatalog) mergeRuntimeStorageResourcesJSONEnv(envName string) error {
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		return nil
	}
	var parsed []RuntimeStorageMountResource
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return fmt.Errorf("invalid %s runtime storage resources JSON: %w", envName, err)
	}
	if err := validateRuntimeMountCatalogResources(envName, nil, parsed); err != nil {
		return err
	}
	for _, item := range parsed {
		catalog.addStorageResource(item)
	}
	return nil
}

func validateRuntimeMountCatalogResources(envName string, secrets []RuntimeSecretMountResource, storage []RuntimeStorageMountResource) error {
	for _, item := range secrets {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = "<missing>"
		}
		if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.SecretProviderClass) == "" {
			return fmt.Errorf("%s secret resource %s requires id and secretProviderClass", envName, id)
		}
	}
	for _, item := range storage {
		kind := strings.TrimSpace(item.Kind)
		if kind == "" {
			kind = runtimeStorageKindEmptyDir
		}
		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = "<missing>"
		}
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("%s storage resource %s requires id", envName, id)
		}
		if kind != runtimeStorageKindEmptyDir && kind != runtimeStorageKindPVC {
			return fmt.Errorf("%s storage resource %s has unsupported kind %q", envName, item.ID, kind)
		}
		if kind == runtimeStorageKindPVC && strings.TrimSpace(item.PVCName) == "" {
			return fmt.Errorf("%s pvc storage resource %s requires pvcName", envName, item.ID)
		}
	}
	return nil
}

func (catalog *RuntimeMountCatalog) addSecretResource(resource RuntimeSecretMountResource) {
	resource.ID = strings.TrimSpace(resource.ID)
	resource.SecretProviderClass = strings.TrimSpace(resource.SecretProviderClass)
	if resource.ID == "" || resource.SecretProviderClass == "" {
		return
	}
	resource.Name = strings.TrimSpace(resource.Name)
	if resource.Name == "" {
		resource.Name = resource.ID
	}
	resource.Kind = strings.TrimSpace(resource.Kind)
	if resource.Kind == "" {
		resource.Kind = runtimeSecretKindSecretProviderClass
	}
	resource.DefaultMountPath = cleanMountPathOrDefault(resource.DefaultMountPath, "/mnt/secrets")
	resource.ReadOnly = true
	for i, item := range catalog.Secrets {
		if item.ID == resource.ID || item.SecretProviderClass == resource.SecretProviderClass { // pragma: allowlist secret
			catalog.Secrets[i] = resource
			return
		}
	}
	catalog.Secrets = append(catalog.Secrets, resource)
}

func (catalog *RuntimeMountCatalog) addStorageResource(resource RuntimeStorageMountResource) {
	resource.ID = strings.TrimSpace(resource.ID)
	if resource.ID == "" {
		return
	}
	resource.Name = strings.TrimSpace(resource.Name)
	if resource.Name == "" {
		resource.Name = resource.ID
	}
	resource.Kind = strings.TrimSpace(resource.Kind)
	if resource.Kind == "" {
		resource.Kind = runtimeStorageKindEmptyDir
	}
	resource.PVCName = strings.TrimSpace(resource.PVCName)
	resource.DefaultMountPath = cleanMountPathOrDefault(resource.DefaultMountPath, defaultRuntimeStorageMountPath)
	if resource.Kind == runtimeStorageKindEmptyDir {
		resource.AllowWrite = true
	}
	for i, item := range catalog.Storage {
		if item.ID == resource.ID {
			catalog.Storage[i] = resource
			return
		}
	}
	catalog.Storage = append(catalog.Storage, resource)
}

func runtimeMountTargetIDsForEnv(envName string) []string {
	if value := strings.TrimSpace(os.Getenv(envName + "_TARGET_IDS")); value != "" {
		return splitCSV(value)
	}
	return splitCSV(os.Getenv("PIPELINE_RUNTIME_MOUNT_TARGET_IDS"))
}

func (catalog RuntimeMountCatalog) normalized() RuntimeMountCatalog {
	out := RuntimeMountCatalog{
		Secrets: append([]RuntimeSecretMountResource(nil), catalog.Secrets...),
		Storage: append([]RuntimeStorageMountResource(nil), catalog.Storage...),
	}
	if out.Secrets == nil { // pragma: allowlist secret
		out.Secrets = []RuntimeSecretMountResource{}
	}
	if out.Storage == nil {
		out.Storage = []RuntimeStorageMountResource{}
	}
	for i := range out.Secrets {
		out.Secrets[i].ID = strings.TrimSpace(out.Secrets[i].ID)
		out.Secrets[i].Name = strings.TrimSpace(out.Secrets[i].Name)
		out.Secrets[i].Kind = strings.TrimSpace(out.Secrets[i].Kind)
		if out.Secrets[i].Kind == "" {
			out.Secrets[i].Kind = runtimeSecretKindSecretProviderClass
		}
		out.Secrets[i].SecretProviderClass = strings.TrimSpace(out.Secrets[i].SecretProviderClass)
		out.Secrets[i].DefaultMountPath = cleanMountPathOrDefault(out.Secrets[i].DefaultMountPath, "/mnt/secrets")
		out.Secrets[i].ReadOnly = true
	}
	for i := range out.Storage {
		out.Storage[i].ID = strings.TrimSpace(out.Storage[i].ID)
		out.Storage[i].Name = strings.TrimSpace(out.Storage[i].Name)
		out.Storage[i].Kind = strings.TrimSpace(out.Storage[i].Kind)
		if out.Storage[i].Kind == "" {
			out.Storage[i].Kind = runtimeStorageKindEmptyDir
		}
		out.Storage[i].PVCName = strings.TrimSpace(out.Storage[i].PVCName)
		out.Storage[i].DefaultMountPath = cleanMountPathOrDefault(out.Storage[i].DefaultMountPath, defaultRuntimeStorageMountPath)
		if out.Storage[i].Kind == runtimeStorageKindEmptyDir {
			out.Storage[i].AllowWrite = true
		}
	}
	return out
}

func (uc *Usecase) applyRuntimeMounts(pipe *transpiler.Pipeline, target *models.ExecutionTarget) error {
	if pipe == nil {
		return nil
	}
	catalog, err := uc.runtimeMountCatalogOrDefault()
	if err != nil {
		return err
	}
	secrets := make(map[string]RuntimeSecretMountResource, len(catalog.Secrets))
	for _, item := range catalog.Secrets {
		if item.ID != "" {
			secrets[item.ID] = item
		}
	}
	storage := make(map[string]RuntimeStorageMountResource, len(catalog.Storage))
	for _, item := range catalog.Storage {
		if item.ID != "" {
			storage[item.ID] = item
		}
	}
	return applyRuntimeMountsToNodes(pipe.Nodes, target, secrets, storage)
}

func applyRuntimeMountsToNodes(
	nodes []transpiler.Node,
	target *models.ExecutionTarget,
	secrets map[string]RuntimeSecretMountResource,
	storage map[string]RuntimeStorageMountResource,
) error {
	for i := range nodes {
		if err := applyRuntimeMountsToNode(&nodes[i], target, secrets, storage); err != nil {
			return err
		}
		if len(nodes[i].SubNodes) > 0 {
			if err := applyRuntimeMountsToNodes(nodes[i].SubNodes, target, secrets, storage); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyRuntimeMountsToNode(
	node *transpiler.Node,
	target *models.ExecutionTarget,
	secrets map[string]RuntimeSecretMountResource,
	storage map[string]RuntimeStorageMountResource,
) error {
	if node == nil {
		return nil
	}
	seenMountPaths := map[string]string{}
	usedVolumeNames := map[string]struct{}{}
	for _, mount := range node.VolumeMounts {
		if mount.Name != "" {
			usedVolumeNames[mount.Name] = struct{}{}
		}
		if mount.MountPath != "" {
			seenMountPaths[path.Clean(mount.MountPath)] = mount.Name
		}
	}
	seenResources := map[string]struct{}{}
	for index, binding := range node.RuntimeSecrets {
		resourceID := strings.TrimSpace(binding.ResourceID)
		if resourceID == "" {
			return fmt.Errorf("%w: node %s runtime secret resourceId is required", ErrInvalidArgument, node.ID)
		}
		if _, ok := seenResources["secret:"+resourceID]; ok {
			return fmt.Errorf("%w: node %s runtime secret %s is duplicated", ErrInvalidArgument, node.ID, resourceID)
		}
		seenResources["secret:"+resourceID] = struct{}{}
		resource, ok := secrets[resourceID]
		if !ok {
			return fmt.Errorf("%w: node %s runtime secret %s is not available", ErrInvalidArgument, node.ID, resourceID)
		}
		if !runtimeMountAllowedOnTarget(target, resource.TargetIDs) {
			return fmt.Errorf("%w: node %s runtime secret %s is not available on target %s", ErrInvalidArgument, node.ID, resourceID, runtimeMountTargetID(target))
		}
		if resource.SecretProviderClass == "" {
			return fmt.Errorf("%w: runtime secret %s has no SecretProviderClass", ErrInvalidArgument, resourceID)
		}
		mountPath, err := runtimeBindingMountPath(node.ID, resourceID, binding.MountPath, resource.DefaultMountPath, seenMountPaths)
		if err != nil {
			return err
		}
		volumeName := uniqueRuntimeMountVolumeName(usedVolumeNames, "secret", node.ID, resourceID, index)
		node.VolumeMounts = append(node.VolumeMounts, transpiler.VolumeMount{
			Name:                   volumeName,
			MountPath:              mountPath,
			ReadOnly:               true,
			CSISecretProviderClass: resource.SecretProviderClass,
		})
		addOrReplaceNodeEnv(node, runtimeMountEnvName("PIPELINE_SECRET", resourceID), mountPath)
	}
	for index, binding := range node.StorageMounts {
		resourceID := strings.TrimSpace(binding.ResourceID)
		if resourceID == "" {
			return fmt.Errorf("%w: node %s storage mount resourceId is required", ErrInvalidArgument, node.ID)
		}
		if _, ok := seenResources["storage:"+resourceID]; ok {
			return fmt.Errorf("%w: node %s storage mount %s is duplicated", ErrInvalidArgument, node.ID, resourceID)
		}
		seenResources["storage:"+resourceID] = struct{}{}
		resource, ok := storage[resourceID]
		if !ok {
			return fmt.Errorf("%w: node %s storage mount %s is not available", ErrInvalidArgument, node.ID, resourceID)
		}
		if !runtimeMountAllowedOnTarget(target, resource.TargetIDs) {
			return fmt.Errorf("%w: node %s storage mount %s is not available on target %s", ErrInvalidArgument, node.ID, resourceID, runtimeMountTargetID(target))
		}
		mountPath, err := runtimeBindingMountPath(node.ID, resourceID, binding.MountPath, resource.DefaultMountPath, seenMountPaths)
		if err != nil {
			return err
		}
		readOnly := resource.ReadOnly
		if binding.ReadOnly != nil {
			readOnly = *binding.ReadOnly
		}
		if !readOnly && !resource.AllowWrite {
			return fmt.Errorf("%w: node %s storage mount %s does not support write mode", ErrInvalidArgument, node.ID, resourceID)
		}
		volumeName := uniqueRuntimeMountVolumeName(usedVolumeNames, "storage", node.ID, resourceID, index)
		switch resource.Kind {
		case runtimeStorageKindEmptyDir:
			node.VolumeMounts = append(node.VolumeMounts, transpiler.VolumeMount{
				Name:      volumeName,
				MountPath: mountPath,
				ReadOnly:  readOnly,
				EmptyDir:  true,
			})
		case runtimeStorageKindPVC:
			if resource.PVCName == "" {
				return fmt.Errorf("%w: storage mount %s has no PVC name", ErrInvalidArgument, resourceID)
			}
			node.VolumeMounts = append(node.VolumeMounts, transpiler.VolumeMount{
				Name:      volumeName,
				MountPath: mountPath,
				ReadOnly:  readOnly,
				PVCName:   resource.PVCName,
			})
		default:
			return fmt.Errorf("%w: storage mount %s has unsupported kind %q", ErrInvalidArgument, resourceID, resource.Kind)
		}
		addOrReplaceNodeEnv(node, runtimeMountEnvName("PIPELINE_STORAGE", resourceID), mountPath)
	}
	return nil
}

func runtimeBindingMountPath(nodeID, resourceID, requestedPath, defaultPath string, seen map[string]string) (string, error) {
	mountPath := cleanMountPathOrDefault(requestedPath, defaultPath)
	if err := validateRuntimeMountPath(nodeID, resourceID, mountPath); err != nil {
		return "", err
	}
	if prior, ok := seen[mountPath]; ok {
		if prior == "" {
			prior = "existing mount"
		}
		return "", fmt.Errorf("%w: node %s mount path %s conflicts with %s", ErrInvalidArgument, nodeID, mountPath, prior)
	}
	seen[mountPath] = resourceID
	return mountPath, nil
}

func validateRuntimeMountPath(nodeID, resourceID, mountPath string) error {
	if mountPath == "" || !strings.HasPrefix(mountPath, "/") {
		return fmt.Errorf("%w: node %s mount %s path must be absolute", ErrInvalidArgument, nodeID, resourceID)
	}
	for _, segment := range strings.Split(mountPath, "/") {
		if segment == ".." {
			return fmt.Errorf("%w: node %s mount %s path cannot contain ..", ErrInvalidArgument, nodeID, resourceID)
		}
	}
	cleaned := path.Clean(mountPath)
	if cleaned == "/" {
		return fmt.Errorf("%w: node %s mount %s path cannot be /", ErrInvalidArgument, nodeID, resourceID)
	}
	blocked := []string{
		"/tmp/outputs",
		"/var/run/secrets/kubernetes.io",
		"/proc",
		"/sys",
		"/dev",
	}
	for _, prefix := range blocked {
		if cleaned == prefix || strings.HasPrefix(cleaned, prefix+"/") {
			return fmt.Errorf("%w: node %s mount %s path %s is reserved", ErrInvalidArgument, nodeID, resourceID, cleaned)
		}
	}
	return nil
}

var invalidRuntimeMountNameChars = regexp.MustCompile(`[^a-z0-9-]+`)
var invalidRuntimeMountEnvChars = regexp.MustCompile(`[^A-Za-z0-9]+`)

func uniqueRuntimeMountVolumeName(used map[string]struct{}, prefix, nodeID, resourceID string, index int) string {
	base := fmt.Sprintf("%s-%s-%s-%d", prefix, nodeID, resourceID, index+1)
	value := invalidRuntimeMountNameChars.ReplaceAllString(strings.ToLower(base), "-")
	value = strings.Trim(value, "-")
	if value == "" {
		value = prefix
	}
	if len(value) > 63 {
		value = strings.Trim(value[:63], "-")
	}
	candidate := value
	for suffix := 2; ; suffix++ {
		if _, ok := used[candidate]; !ok {
			used[candidate] = struct{}{}
			return candidate
		}
		trimLen := 63 - len(strconv.Itoa(suffix)) - 1
		if trimLen < 1 {
			trimLen = len(prefix)
		}
		candidate = fmt.Sprintf("%s-%d", strings.Trim(value[:min(len(value), trimLen)], "-"), suffix)
	}
}

func runtimeMountEnvName(prefix, resourceID string) string {
	suffix := invalidRuntimeMountEnvChars.ReplaceAllString(strings.ToUpper(resourceID), "_")
	suffix = strings.Trim(suffix, "_")
	if suffix == "" {
		suffix = "MOUNT"
	}
	return prefix + "_" + suffix + "_PATH"
}

func addOrReplaceNodeEnv(node *transpiler.Node, name, value string) {
	for i := range node.Component.Env {
		if node.Component.Env[i].Name == name {
			node.Component.Env[i].Value = value
			node.Component.Env[i].From = ""
			return
		}
	}
	node.Component.Env = append(node.Component.Env, transpiler.EnvVar{Name: name, Value: value})
}

func runtimeMountAllowedOnTarget(target *models.ExecutionTarget, targetIDs []string) bool {
	if len(targetIDs) == 0 {
		return true
	}
	names := runtimeMountTargetNames(target)
	for _, targetID := range targetIDs {
		if _, ok := names[strings.TrimSpace(targetID)]; ok {
			return true
		}
	}
	return false
}

func runtimeMountTargetNames(target *models.ExecutionTarget) map[string]struct{} {
	names := map[string]struct{}{
		runtimeMountTargetID(target): {},
	}
	if target == nil {
		return names
	}
	for _, value := range []string{target.Name, target.Namespace} {
		if value = strings.TrimSpace(value); value != "" {
			names[value] = struct{}{}
		}
	}
	return names
}

func runtimeMountTargetID(target *models.ExecutionTarget) string {
	if target == nil || strings.TrimSpace(target.ID) == "" {
		return "default"
	}
	return strings.TrimSpace(target.ID)
}

func cleanMountPathOrDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = strings.TrimSpace(fallback)
	}
	if value == "" {
		return ""
	}
	return path.Clean(value)
}

func getenvDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func parseBoolEnv(key string) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return false
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return parsed
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			out = append(out, item)
		}
	}
	return out
}
