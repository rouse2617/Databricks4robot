package algorun

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	algorunUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/algorun"
)

// Handler exposes algo_runs HTTP API.
type Handler struct {
	uc *algorunUC.Usecase
}

func New(uc *algorunUC.Usecase) *Handler {
	return &Handler{uc: uc}
}

// Create registers a new algo run.
// @Summary      Create algo run
// @Tags         algo-runs
// @Accept       json
// @Produce      json
// @Param        body body algorunUC.CreateInput true "Create algo run"
// @Success      201 {object} models.AlgoRun
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      409 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /algo-runs [post]
func (h *Handler) Create(c *gin.Context) {
	var req algorunUC.CreateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	run, err := h.uc.Create(c.Request.Context(), req)
	if err != nil {
		mapAlgoRunErr(c, err)
		return
	}
	c.JSON(201, run)
}

// Start transitions pending → running.
// @Summary      Start algo run
// @Tags         algo-runs
// @Produce      json
// @Param        run_id path string true "Run ID"
// @Success      200 {object} models.AlgoRun
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      404 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /algo-runs/{run_id}/start [post]
func (h *Handler) Start(c *gin.Context) {
	runID := strings.TrimSpace(c.Param("run_id"))
	run, err := h.uc.Start(c.Request.Context(), runID)
	if err != nil {
		mapAlgoRunErr(c, err)
		return
	}
	c.JSON(200, run)
}

// Finish transitions running → ok|failed.
// @Summary      Finish algo run
// @Tags         algo-runs
// @Accept       json
// @Produce      json
// @Param        run_id path string true "Run ID"
// @Param        body body algorunUC.FinishInput true "Finish algo run"
// @Success      200 {object} models.AlgoRun
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      404 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /algo-runs/{run_id}/finish [post]
func (h *Handler) Finish(c *gin.Context) {
	runID := strings.TrimSpace(c.Param("run_id"))
	var req algorunUC.FinishInput
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	run, err := h.uc.Finish(c.Request.Context(), runID, req)
	if err != nil {
		mapAlgoRunErr(c, err)
		return
	}
	c.JSON(200, run)
}

// Get returns one algo run.
// @Summary      Get algo run
// @Tags         algo-runs
// @Produce      json
// @Param        run_id path string true "Run ID"
// @Success      200 {object} models.AlgoRun
// @Failure      404 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /algo-runs/{run_id} [get]
func (h *Handler) Get(c *gin.Context) {
	runID := strings.TrimSpace(c.Param("run_id"))
	run, err := h.uc.Get(c.Request.Context(), runID)
	if err != nil {
		mapAlgoRunErr(c, err)
		return
	}
	c.JSON(200, run)
}

// List returns algo runs with optional filters.
// @Summary      List algo runs
// @Tags         algo-runs
// @Produce      json
// @Param        algo_name query string false "Filter by algo_name"
// @Param        status query string false "Filter by status"
// @Param        started_after query string false "RFC3339 lower bound on started_at"
// @Param        started_before query string false "RFC3339 upper bound on started_at"
// @Param        page query int false "Page number (default 1)"
// @Param        page_size query int false "Page size (default 50, max 200)"
// @Success      200 {object} object
// @Security     GraceToken
// @Router       /algo-runs [get]
func (h *Handler) List(c *gin.Context) {
	f := algorunUC.ListFilter{
		AlgoName: c.Query("algo_name"),
		Status:   c.Query("status"),
		Page:     1,
		PageSize: 50,
	}
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Page = n
		}
	}
	if v := c.Query("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.PageSize = n
		}
	}
	if v := c.Query("started_after"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.StartedAfter = &t
		}
	}
	if v := c.Query("started_before"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.StartedBefore = &t
		}
	}
	runs, total, err := h.uc.List(c.Request.Context(), f)
	if err != nil {
		mapAlgoRunErr(c, err)
		return
	}
	if runs == nil {
		runs = []*models.AlgoRun{} //nolint:staticcheck // ensure JSON []
	}
	c.JSON(200, gin.H{
		"items":     runs,
		"total":     total,
		"page":      f.Page,
		"page_size": f.PageSize,
	})
}

// Cancel cancels a pending or running algo run.
// @Summary      Cancel algo run
// @Tags         algo-runs
// @Accept       json
// @Produce      json
// @Param        run_id path string true "Run ID"
// @Param        body body CancelRequest true "Cancel reason"
// @Success      200 {object} models.AlgoRun
// @Failure      400 {object} httpresp.ErrorBody
// @Failure      404 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /algo-runs/{run_id}/cancel [post]
func (h *Handler) Cancel(c *gin.Context) {
	runID := strings.TrimSpace(c.Param("run_id"))
	var req CancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", map[string]any{"error": err.Error()})
		return
	}
	run, err := h.uc.Cancel(c.Request.Context(), runID, req.Reason)
	if err != nil {
		mapAlgoRunErr(c, err)
		return
	}
	c.JSON(200, run)
}

// GetAffectedAssets returns assets processed by an algo run.
// @Summary      Get affected assets
// @Tags         algo-runs
// @Produce      json
// @Param        run_id path string true "Run ID"
// @Success      200 {array} repository.AffectedAsset
// @Failure      404 {object} httpresp.ErrorBody
// @Security     GraceToken
// @Router       /algo-runs/{run_id}/affected-assets [get]
func (h *Handler) GetAffectedAssets(c *gin.Context) {
	runID := strings.TrimSpace(c.Param("run_id"))
	assets, err := h.uc.GetAffectedAssets(c.Request.Context(), runID)
	if err != nil {
		mapAlgoRunErr(c, err)
		return
	}
	if assets == nil {
		assets = []*repository.AffectedAsset{} //nolint:staticcheck // ensure JSON []
	}
	c.JSON(200, assets)
}

// CancelRequest is the body for POST /algo-runs/:run_id/cancel.
type CancelRequest struct {
	Reason string `json:"reason" binding:"required"`
}

func mapAlgoRunErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, algorunUC.ErrInvalidRunID),
		errors.Is(err, algorunUC.ErrInvalidAlgoKind),
		errors.Is(err, algorunUC.ErrInvalidStatus),
		errors.Is(err, algorunUC.ErrMissingField):
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
	case errors.Is(err, repository.ErrAlgoRunNotFound), errors.Is(err, algorunUC.ErrRunNotFound):
		httpresp.NotFound(c, httpresp.CodeAlgoRunNotFound, err.Error())
	case errors.Is(err, repository.ErrAlgoRunBadState), errors.Is(err, algorunUC.ErrBadTransition):
		httpresp.BadRequest(c, httpresp.CodeInvalidState, err.Error(), nil)
	case errors.Is(err, repository.ErrDuplicateRunID):
		httpresp.Conflict(c, httpresp.CodeInvalidArgument, "run_id already exists", nil)
	default:
		httpresp.Internal(c, err.Error())
	}
}
