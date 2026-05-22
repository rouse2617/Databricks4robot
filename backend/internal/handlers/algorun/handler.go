package algorun

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
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
