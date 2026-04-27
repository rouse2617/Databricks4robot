package registry

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"data-platform/internal/config"
)

// Handler serves read-only registry endpoints (algo-registry, tag-registry).
type Handler struct {
	algoRegistry *config.AlgoRegistry
	tagRegistry  *config.TagRegistry
}

// New creates a new registry handler.
func New(algoRegistry *config.AlgoRegistry, tagRegistry *config.TagRegistry) *Handler {
	return &Handler{algoRegistry: algoRegistry, tagRegistry: tagRegistry}
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
	Key       string   `json:"key"`
	Type      string   `json:"type"`
	Values    []string `json:"values,omitempty"`
	MaxLength int      `json:"max_length,omitempty"`
}

// TagRegistry returns the list of registered tags.
// GET /api/v1/tag-registry
func (h *Handler) TagRegistry(c *gin.Context) {
	tags := h.tagRegistry.GetAllTags()
	var items []TagRegistryItem
	for key, def := range tags {
		item := TagRegistryItem{
			Key:  key,
			Type: def.Type,
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
