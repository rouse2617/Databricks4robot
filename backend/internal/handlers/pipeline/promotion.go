package pipeline

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	pipelineUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/pipeline"
)

func requirePipelineAdmin(c *gin.Context) bool {
	role, _ := c.Get(middleware.CtxKeyRole)
	roleText, _ := role.(string)
	if roleText == "admin" || middleware.GetUserEmail(c) == "sdk" {
		return true
	}
	httpresp.Error(c, http.StatusForbidden, httpresp.CodeUnauthorized, "admin role is required", nil)
	return false
}

// PromotionPlan handles POST /api/v1/pipelines/:id/promotion-plan.
func (h *Handler) PromotionPlan(c *gin.Context) {
	if !requirePipelineAdmin(c) {
		return
	}
	var body struct {
		Target   string                                       `json:"target"`
		Mappings []models.PipelinePromotionMappingRequirement `json:"mappings"`
	}
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", nil)
			return
		}
	}
	if body.Target != "" && body.Target != "prod" {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "target must be prod", nil)
		return
	}
	plan, err := h.uc.BuildPromotionPlan(c.Request.Context(), c.Param("id"), body.Mappings)
	if err != nil {
		writePromotionError(c, err)
		return
	}
	c.JSON(http.StatusOK, plan)
}

// PromoteToProd copies a dev template to the prod scope.
func (h *Handler) PromoteToProd(c *gin.Context) {
	if !requirePipelineAdmin(c) {
		return
	}
	result, err := h.uc.Promote(c.Request.Context(), c.Param("id"))
	if err != nil {
		writePromotionError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func writePromotionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, pipelineUC.ErrTemplateNotFound):
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
	case errors.Is(err, pipelineUC.ErrInvalidArgument):
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
	case errors.Is(err, pipelineUC.ErrVersionConflict):
		httpresp.Error(c, http.StatusConflict, httpresp.CodeInvalidArgument, err.Error(), nil)
	case errors.Is(err, pipelineUC.ErrPromotionNotReady):
		httpresp.Error(c, http.StatusUnprocessableEntity, httpresp.CodeInvalidArgument, err.Error(), nil)
	default:
		httpresp.Internal(c, err.Error())
	}
}
