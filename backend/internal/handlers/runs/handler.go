package runs

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	runsUC "github.com/CyberOrigin2077/cyber-databrew/internal/usecase/runs"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
)

type Handler struct {
	uc *runsUC.Usecase
}

func New(uc *runsUC.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) ListRuns(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	result, err := h.uc.ListRuns(c.Request.Context(), models.DatabrewRunListFilter{
		Type:     strings.TrimSpace(c.Query("type")),
		Status:   strings.TrimSpace(c.Query("status")),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetRun(c *gin.Context) {
	id := strings.TrimSpace(c.Param("runId"))
	run, err := h.uc.GetRun(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, runsUC.ErrRunNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, enrichRunResponse(run))
}

func (h *Handler) GetRunByWorkflowName(c *gin.Context) {
	name := strings.TrimSpace(c.Param("workflowName"))
	run, err := h.uc.GetRunByWorkflowName(c.Request.Context(), name)
	if err != nil {
		if errors.Is(err, runsUC.ErrRunNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, enrichRunResponse(run))
}

func (h *Handler) GetRuntime(c *gin.Context) {
	id := strings.TrimSpace(c.Param("runId"))
	payload, err := h.uc.GetRuntime(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, runsUC.ErrRunNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, payload)
}

func (h *Handler) GetNodes(c *gin.Context) {
	id := strings.TrimSpace(c.Param("runId"))
	nodes, err := h.uc.GetNodes(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, runsUC.ErrRunNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": nodes})
}

func (h *Handler) GetEvents(c *gin.Context) {
	id := strings.TrimSpace(c.Param("runId"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	cursor, _ := strconv.ParseInt(c.DefaultQuery("cursor", "0"), 10, 64)
	var from *time.Time
	if raw := strings.TrimSpace(c.Query("from")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "from must be RFC3339", nil)
			return
		}
		from = &parsed
	}
	var to *time.Time
	if raw := strings.TrimSpace(c.Query("to")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "to must be RFC3339", nil)
			return
		}
		to = &parsed
	}
	result, err := h.uc.GetEvents(c.Request.Context(), id, models.PipelineRunEventListOptions{
		Limit:       limit,
		Cursor:      cursor,
		SubjectType: strings.TrimSpace(c.Query("subjectType")),
		EventType:   strings.TrimSpace(c.Query("eventType")),
		Status:      strings.TrimSpace(c.Query("status")),
		Query:       strings.TrimSpace(c.Query("q")),
		From:        from,
		To:          to,
	})
	if err != nil {
		if errors.Is(err, runsUC.ErrRunNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetPods(c *gin.Context) {
	id := strings.TrimSpace(c.Param("runId"))
	items, err := h.uc.GetPods(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, runsUC.ErrRunNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) CreateComponentBuild(c *gin.Context) {
	var body struct {
		ComponentID     string `json:"componentId"`
		ComponentName   string `json:"componentName"`
		RepoURL         string `json:"repoUrl"`
		GitRef          string `json:"gitRef"`
		Dockerfile      string `json:"dockerfile"`
		BuildContext    string `json:"buildContext"`
		ImageRepository string `json:"imageRepository"`
		ImageTag        string `json:"imageTag"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", nil)
		return
	}
	run, err := h.uc.CreateComponentBuild(c.Request.Context(), runsUC.CreateComponentBuildInput{
		ComponentID:     strings.TrimSpace(body.ComponentID),
		ComponentName:   strings.TrimSpace(body.ComponentName),
		RepoURL:         strings.TrimSpace(body.RepoURL),
		GitRef:          strings.TrimSpace(body.GitRef),
		Dockerfile:      strings.TrimSpace(body.Dockerfile),
		BuildContext:    strings.TrimSpace(body.BuildContext),
		ImageRepository: strings.TrimSpace(body.ImageRepository),
		ImageTag:        strings.TrimSpace(body.ImageTag),
	})
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	c.JSON(http.StatusCreated, enrichRunResponse(run))
}

func (h *Handler) CreateRAGBuild(c *gin.Context) {
	var body struct {
		KnowledgeBaseID string                 `json:"knowledgeBaseId"`
		EmbeddingModel  string                 `json:"embeddingModel"`
		VectorIndexName string                 `json:"vectorIndexName"`
		ReleaseVersion  string                 `json:"releaseVersion"`
		Datasource      map[string]interface{} `json:"datasource"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid request body", nil)
		return
	}
	run, err := h.uc.CreateRAGBuild(c.Request.Context(), runsUC.CreateRAGBuildInput{
		KnowledgeBaseID: strings.TrimSpace(body.KnowledgeBaseID),
		EmbeddingModel:  strings.TrimSpace(body.EmbeddingModel),
		VectorIndexName: strings.TrimSpace(body.VectorIndexName),
		ReleaseVersion:  strings.TrimSpace(body.ReleaseVersion),
		Datasource:      body.Datasource,
	})
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, err.Error(), nil)
		return
	}
	c.JSON(http.StatusCreated, enrichRunResponse(run))
}

func (h *Handler) GetArtifacts(c *gin.Context) {
	id := strings.TrimSpace(c.Param("runId"))
	items, err := h.uc.GetArtifacts(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, runsUC.ErrRunNotFound) {
			httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
			return
		}
		httpresp.Internal(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) RetryRun(c *gin.Context) {
	id := strings.TrimSpace(c.Param("runId"))
	run, err := h.uc.RetryRun(c.Request.Context(), id)
	if err != nil {
		mapRunError(c, err)
		return
	}
	c.JSON(http.StatusCreated, run)
}

func (h *Handler) StopRun(c *gin.Context) {
	h.runOperation(c, h.uc.StopRun, "run stopped")
}

func (h *Handler) SuspendRun(c *gin.Context) {
	h.runOperation(c, h.uc.SuspendRun, "run suspended")
}

func (h *Handler) ResumeRun(c *gin.Context) {
	h.runOperation(c, h.uc.ResumeRun, "run resumed")
}

func (h *Handler) ResubmitRun(c *gin.Context) {
	h.runOperation(c, h.uc.ResubmitRun, "run resubmitted")
}

func (h *Handler) TerminateRun(c *gin.Context) {
	h.runOperation(c, h.uc.TerminateRun, "run terminated")
}

func (h *Handler) runOperation(c *gin.Context, fn func(context.Context, string) error, message string) {
	id := strings.TrimSpace(c.Param("runId"))
	if err := fn(c.Request.Context(), id); err != nil {
		mapRunError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": message})
}

func mapRunError(c *gin.Context, err error) {
	if errors.Is(err, runsUC.ErrRunNotFound) {
		httpresp.NotFound(c, httpresp.CodeAssetNotFound, err.Error())
		return
	}
	httpresp.Internal(c, err.Error())
}

func enrichRunResponse(run *models.DatabrewRun) gin.H {
	return gin.H{
		"id":          run.ID,
		"type":        run.Type,
		"name":        run.Name,
		"status":      run.Status,
		"statusLabel": run.StatusLabel,
		"runtime": gin.H{
			"type":         run.Runtime,
			"namespace":    run.RuntimeNamespace,
			"resourceName": run.RuntimeResourceName,
			"uid":          run.RuntimeUID,
		},
		"owner":      run.Owner,
		"createdBy":  run.CreatedBy,
		"message":    run.Message,
		"summary":    run.Summary,
		"actions":    run.Actions,
		"createdAt":  run.CreatedAt,
		"startedAt":  run.StartedAt,
		"finishedAt": run.FinishedAt,
		"updatedAt":  run.UpdatedAt,
	}
}
