package registry

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"

	"data-platform/internal/config"
	"data-platform/internal/lifecycle"
)

type Handler struct {
	algoRegistry        *config.AlgoRegistry
	tagRegistry         *config.TagRegistry
	metricRegistry      *config.MetricRegistry
	actionLabelRegistry *config.ActionLabelRegistry
}

// New creates a new registry handler.
func New(
	algoRegistry *config.AlgoRegistry,
	tagRegistry *config.TagRegistry,
	metricRegistry *config.MetricRegistry,
	actionLabelRegistry *config.ActionLabelRegistry,
) *Handler {
	return &Handler{
		algoRegistry:        algoRegistry,
		tagRegistry:         tagRegistry,
		metricRegistry:      metricRegistry,
		actionLabelRegistry: actionLabelRegistry,
	}
}

// AlgoRegistryItem is the JSON shape returned by GET /api/v1/algo-registry.
type AlgoRegistryItem struct {
	Key       string   `json:"key"`
	Name      string   `json:"name"`
	Version   string   `json:"version"`
	DependsOn []string `json:"depends_on"`
}

// AlgoRegistry returns the list of registered algorithms.
// GET /api/v1/algo-registry
func (h *Handler) AlgoRegistry(c *gin.Context) {
	algos := h.algoRegistry.GetAllAlgorithms()
	var items []AlgoRegistryItem
	for name, def := range algos {
		for _, ver := range def.Versions {
			deps := def.DependsOn
			if deps == nil {
				deps = []string{}
			}
			items = append(items, AlgoRegistryItem{
				Key:       name + "@" + ver,
				Name:      name,
				Version:   ver,
				DependsOn: deps,
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// TagRegistryItem is the JSON shape returned by GET /api/v1/tag-registry.
type TagRegistryItem struct {
	Key         string   `json:"key"`
	Description string   `json:"description,omitempty"`
	Type        string   `json:"type"`
	Values      []string `json:"values,omitempty"`
	MaxLength   int      `json:"max_length,omitempty"`
}

// TagRegistry returns the list of registered tags.
// GET /api/v1/tag-registry
func (h *Handler) TagRegistry(c *gin.Context) {
	tags := h.tagRegistry.GetAllTags()
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var items []TagRegistryItem
	for _, key := range keys {
		def := tags[key]
		item := TagRegistryItem{
			Key:         key,
			Description: def.Description,
			Type:        def.Type,
		}
		if def.Type == "enum" {
			vals := def.Values
			if vals == nil {
				vals = []string{}
			}
			item.Values = vals
		}
		if def.MaxLength > 0 {
			item.MaxLength = def.MaxLength
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// LifecycleStates returns asset lifecycle_state values allowed by PostgreSQL
// (chk_lifecycle_state). Same order as internal/lifecycle.AllowedAssetLifecycleStates.
// GET /api/v1/lifecycle-states
func (h *Handler) LifecycleStates(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": lifecycle.AllowedAssetLifecycleStates})
}

// MetricRegistry returns metric definitions from metric_registry.yaml.
// GET /api/v1/metric-registry
func (h *Handler) MetricRegistry(c *gin.Context) {
	if h.metricRegistry == nil {
		c.JSON(http.StatusOK, gin.H{"items": []config.MetricDefinition{}})
		return
	}
	items := h.metricRegistry.All()
	sort.Slice(items, func(i, j int) bool { return items[i].Key < items[j].Key })
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// ActionLabelRegistry returns action label constraints from action_label_registry.yaml.
// GET /api/v1/action-label-registry
func (h *Handler) ActionLabelRegistry(c *gin.Context) {
	if h.actionLabelRegistry == nil {
		c.JSON(http.StatusOK, gin.H{
			"primary_labels": []string{},
			"labels":         []string{},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"primary_labels": h.actionLabelRegistry.PrimaryLabels(),
		"labels":         h.actionLabelRegistry.Labels(),
	})
}
