package admin

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	espkg "github.com/CyberOrigin2077/cyber-databrew/internal/elasticsearch"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/httpresp"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

const (
	defaultReindexJobsListLimit = 20
	maxReindexJobsListLimit     = 100
	reindexProgressCheckpointN  = 25
	reindexStopForcePauseAfter  = 20 * time.Second
	reindexRunningStaleAfter    = 2 * time.Minute
	reindexPausedExpireAfter    = 7 * 24 * time.Hour
)

type SearchReindexJobView struct {
	ID                    string                            `json:"id"`
	Status                repository.SearchReindexJobStatus `json:"status"`
	DryRun                bool                              `json:"dry_run"`
	PageSize              int                               `json:"page_size"`
	NextPage              int                               `json:"next_page"`
	StopRequested         bool                              `json:"stop_requested"`
	TotalAssets           int64                             `json:"total_assets"`
	AssetsScanned         int64                             `json:"assets_scanned"`
	DocumentsIndexed      int64                             `json:"documents_indexed"`
	DocumentsDeleted      int64                             `json:"documents_deleted"`
	Failed                int64                             `json:"failed"`
	Error                 string                            `json:"error,omitempty"`
	ErrorSamples          []string                          `json:"error_samples,omitempty"`
	ElasticsearchDocCount *int64                            `json:"elasticsearch_doc_count,omitempty"`
	ProgressPct           float64                           `json:"progress_pct"`
	CreatedAt             time.Time                         `json:"created_at"`
	UpdatedAt             time.Time                         `json:"updated_at"`
	StartedAt             *time.Time                        `json:"started_at,omitempty"`
	FinishedAt            *time.Time                        `json:"finished_at,omitempty"`
}

type SearchReindexJobsListResponse struct {
	Items []SearchReindexJobView `json:"items"`
}

func (h *Handler) reconcileReindexJobState(ctx context.Context, job *repository.SearchReindexJob) *repository.SearchReindexJob {
	if h.jobs == nil || job == nil || job.Status != repository.SearchReindexJobStatusRunning {
		return job
	}
	idleFor := time.Since(job.UpdatedAt)
	if idleFor < 0 {
		idleFor = 0
	}
	if job.StopRequested && idleFor >= reindexStopForcePauseAfter {
		_ = h.jobs.MarkPaused(ctx, job.ID)
		if refreshed, err := h.jobs.Get(ctx, job.ID); err == nil && refreshed != nil {
			return refreshed
		}
		return job
	}
	if !job.StopRequested && idleFor >= reindexRunningStaleAfter {
		msg := fmt.Sprintf("reindex runner heartbeat stale for %s; mark failed for manual resume", idleFor.Truncate(time.Second))
		samples := appendReindexError(job.ErrorSamples, msg)
		_ = h.jobs.MarkFailed(ctx, job.ID, msg, samples)
		if refreshed, err := h.jobs.Get(ctx, job.ID); err == nil && refreshed != nil {
			return refreshed
		}
	}
	return job
}

func toSearchReindexJobView(job *repository.SearchReindexJob) SearchReindexJobView {
	progress := 0.0
	if job.TotalAssets > 0 {
		progress = (float64(job.AssetsScanned) / float64(job.TotalAssets)) * 100.0
	}
	if progress < 0 {
		progress = 0
	}
	progress = math.Min(100, progress)
	return SearchReindexJobView{
		ID:                    job.ID,
		Status:                job.Status,
		DryRun:                job.DryRun,
		PageSize:              job.PageSize,
		NextPage:              job.NextPage,
		StopRequested:         job.StopRequested,
		TotalAssets:           job.TotalAssets,
		AssetsScanned:         job.AssetsScanned,
		DocumentsIndexed:      job.DocumentsIndexed,
		DocumentsDeleted:      job.DocumentsDeleted,
		Failed:                job.Failed,
		Error:                 job.Error,
		ErrorSamples:          job.ErrorSamples,
		ElasticsearchDocCount: job.ElasticsearchDocCount,
		ProgressPct:           progress,
		CreatedAt:             job.CreatedAt,
		UpdatedAt:             job.UpdatedAt,
		StartedAt:             job.StartedAt,
		FinishedAt:            job.FinishedAt,
	}
}

// findActiveJob returns the most recent queued/running job (after reconciliation), or nil.
func (h *Handler) findActiveJob(ctx context.Context) (*repository.SearchReindexJob, error) {
	jobs, err := h.jobs.ListRecent(ctx, 5)
	if err != nil {
		return nil, err
	}
	for _, job := range jobs {
		job = h.reconcileReindexJobState(ctx, job)
		if job == nil {
			continue
		}
		if job.Status == repository.SearchReindexJobStatusQueued ||
			job.Status == repository.SearchReindexJobStatusRunning {
			return job, nil
		}
	}
	return nil, nil
}

// SearchReindexCreateJob creates an async PG->ES reindex job and starts it in background.
func (h *Handler) SearchReindexCreateJob(c *gin.Context) {
	if h.jobs == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable,
			"reindex jobs repository not configured", nil)
		return
	}
	if h.es == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable,
			"Elasticsearch is not configured", nil)
		return
	}
	req, err := parseSearchReindexRequest(c)
	if err != nil {
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "invalid reindex request", map[string]any{"error": err.Error()})
		return
	}
	// Reject if there is already an active (queued/running, after reconcile) job to avoid
	// concurrent reindex storms when multiple tabs/users click the button.
	if existing, err := h.findActiveJob(c.Request.Context()); err != nil {
		httpresp.Error(c, http.StatusInternalServerError, httpresp.CodeInternalError, err.Error(), nil)
		return
	} else if existing != nil {
		httpresp.Error(c, http.StatusConflict, httpresp.CodeInvalidArgument,
			"another reindex job is already active; stop it before creating a new one",
			map[string]any{"active_job": toSearchReindexJobView(existing)})
		return
	}
	job, err := h.jobs.Create(c.Request.Context(), req.DryRun, req.PageSize)
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, httpresp.CodeInternalError, err.Error(), nil)
		return
	}
	h.ensureJobRunner(job.ID)
	c.JSON(http.StatusAccepted, toSearchReindexJobView(job))
}

// SearchReindexGetJob returns one async reindex job detail.
func (h *Handler) SearchReindexGetJob(c *gin.Context) {
	if h.jobs == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable,
			"reindex jobs repository not configured", nil)
		return
	}
	jobID := c.Param("id")
	job, err := h.jobs.Get(c.Request.Context(), jobID)
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, httpresp.CodeInternalError, err.Error(), nil)
		return
	}
	if job == nil {
		httpresp.Error(c, http.StatusNotFound, httpresp.CodeInvalidArgument, "reindex job not found", nil)
		return
	}
	job = h.reconcileReindexJobState(c.Request.Context(), job)
	c.JSON(http.StatusOK, toSearchReindexJobView(job))
}

// SearchReindexListJobs returns recent async reindex jobs.
func (h *Handler) SearchReindexListJobs(c *gin.Context) {
	if h.jobs == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable,
			"reindex jobs repository not configured", nil)
		return
	}
	limit := defaultReindexJobsListLimit
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > maxReindexJobsListLimit {
		limit = maxReindexJobsListLimit
	}
	jobs, err := h.jobs.ListRecent(c.Request.Context(), limit)
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, httpresp.CodeInternalError, err.Error(), nil)
		return
	}
	items := make([]SearchReindexJobView, 0, len(jobs))
	now := time.Now()
	for _, job := range jobs {
		job = h.reconcileReindexJobState(c.Request.Context(), job)
		if job.Status == repository.SearchReindexJobStatusPaused &&
			!job.UpdatedAt.IsZero() &&
			now.Sub(job.UpdatedAt) >= reindexPausedExpireAfter {
			_ = h.jobs.MarkAbandoned(c.Request.Context(), job.ID)
			if refreshed, err := h.jobs.Get(c.Request.Context(), job.ID); err == nil && refreshed != nil {
				job = refreshed
			}
		}
		items = append(items, toSearchReindexJobView(job))
	}
	c.JSON(http.StatusOK, SearchReindexJobsListResponse{Items: items})
}

// SearchReindexStopJob requests the job to stop at the next safe checkpoint.
func (h *Handler) SearchReindexStopJob(c *gin.Context) {
	if h.jobs == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable,
			"reindex jobs repository not configured", nil)
		return
	}
	jobID := c.Param("id")
	job, err := h.jobs.RequestStop(c.Request.Context(), jobID)
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, httpresp.CodeInternalError, err.Error(), nil)
		return
	}
	if job == nil {
		httpresp.Error(c, http.StatusNotFound, httpresp.CodeInvalidArgument, "reindex job not found", nil)
		return
	}
	job = h.reconcileReindexJobState(c.Request.Context(), job)
	// queued jobs may not have been claimed yet; pause immediately for deterministic UX.
	if job.Status == repository.SearchReindexJobStatusQueued {
		if err := h.jobs.MarkPaused(c.Request.Context(), job.ID); err == nil {
			job, _ = h.jobs.Get(c.Request.Context(), job.ID)
		}
	}
	if job == nil {
		httpresp.Error(c, http.StatusNotFound, httpresp.CodeInvalidArgument, "reindex job not found", nil)
		return
	}
	c.JSON(http.StatusAccepted, toSearchReindexJobView(job))
}

// SearchReindexResumeJob resumes a paused/failed async reindex job from checkpoint.
func (h *Handler) SearchReindexResumeJob(c *gin.Context) {
	if h.jobs == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable,
			"reindex jobs repository not configured", nil)
		return
	}
	jobID := c.Param("id")
	job, err := h.jobs.Resume(c.Request.Context(), jobID)
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, httpresp.CodeInternalError, err.Error(), nil)
		return
	}
	if job == nil {
		current, getErr := h.jobs.Get(c.Request.Context(), jobID)
		if getErr != nil {
			httpresp.Error(c, http.StatusInternalServerError, httpresp.CodeInternalError, getErr.Error(), nil)
			return
		}
		if current == nil {
			httpresp.Error(c, http.StatusNotFound, httpresp.CodeInvalidArgument, "reindex job not found", nil)
			return
		}
		httpresp.BadRequest(c, httpresp.CodeInvalidArgument, "job is not resumable", map[string]any{"status": current.Status})
		return
	}
	h.ensureJobRunner(job.ID)
	c.JSON(http.StatusAccepted, toSearchReindexJobView(job))
}

// SearchReindexAbandonJob marks a paused reindex job as abandoned.
func (h *Handler) SearchReindexAbandonJob(c *gin.Context) {
	if h.jobs == nil {
		httpresp.Error(c, http.StatusServiceUnavailable, httpresp.CodeServiceUnavailable,
			"reindex jobs repository not configured", nil)
		return
	}
	jobID := c.Param("id")
	if err := h.jobs.MarkAbandoned(c.Request.Context(), jobID); err != nil {
		httpresp.Error(c, http.StatusInternalServerError, httpresp.CodeInternalError, err.Error(), nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "abandoned"})
}

func (h *Handler) ensureJobRunner(jobID string) {
	h.jobMu.Lock()
	if _, exists := h.runningJobs[jobID]; exists {
		h.jobMu.Unlock()
		return
	}
	h.runningJobs[jobID] = struct{}{}
	h.jobMu.Unlock()

	go h.runReindexJob(jobID)
}

func (h *Handler) clearRunningJob(jobID string) {
	h.jobMu.Lock()
	delete(h.runningJobs, jobID)
	h.jobMu.Unlock()
}

func (h *Handler) runReindexJob(jobID string) {
	defer h.clearRunningJob(jobID)
	if h.jobs == nil || h.es == nil || h.assets == nil || h.indexer == nil {
		return
	}
	ctx := context.Background()
	job, err := h.jobs.ClaimForRun(ctx, jobID)
	if err != nil || job == nil {
		return
	}

	page := job.NextPage
	if page <= 0 {
		page = 1
	}
	pageSize := job.PageSize
	if pageSize <= 0 || pageSize > 500 {
		pageSize = 200
	}
	progress := repository.SearchReindexJobProgress{
		TotalAssets:      job.TotalAssets,
		NextPage:         page,
		AssetsScanned:    job.AssetsScanned,
		DocumentsIndexed: job.DocumentsIndexed,
		DocumentsDeleted: job.DocumentsDeleted,
		Failed:           job.Failed,
		ErrorSamples:     append([]string(nil), job.ErrorSamples...),
		IndexCleared:     job.IndexCleared,
	}
	failJob := func(err error) {
		msg := err.Error()
		progress.ErrorSamples = appendReindexError(progress.ErrorSamples, msg)
		_ = h.jobs.MarkFailed(ctx, jobID, msg, progress.ErrorSamples)
	}

	if !job.DryRun && !progress.IndexCleared {
		cleared, err := h.es.DeleteAllDocuments(ctx)
		if err != nil {
			failJob(err)
			return
		}
		progress.DocumentsDeleted += cleared
		progress.IndexCleared = true
		if err := h.jobs.UpdateProgress(ctx, jobID, progress); err != nil {
			failJob(err)
			return
		}
	}

	for {
		stopRequested, err := h.jobs.IsStopRequested(ctx, jobID)
		if err != nil {
			failJob(err)
			return
		}
		if stopRequested {
			_ = h.jobs.MarkPaused(ctx, jobID)
			return
		}

		assets, total, err := h.assets.ListWithFilters(ctx, "", nil, page, pageSize, filter.OrderByClause{SQL: "asset_id ASC"})
		if err != nil {
			failJob(err)
			return
		}
		progress.TotalAssets = total
		if len(assets) == 0 {
			break
		}

		var docs []espkg.BulkIndexDoc
		scannedSinceCheckpoint := int64(0)
		for _, a := range assets {
			progress.AssetsScanned++
			scannedSinceCheckpoint++
			if a == nil || a.AssetID == "" {
				if scannedSinceCheckpoint >= reindexProgressCheckpointN {
					if err := h.jobs.UpdateProgress(ctx, jobID, progress); err != nil {
						failJob(err)
						return
					}
					stopRequested, err = h.jobs.IsStopRequested(ctx, jobID)
					if err != nil {
						failJob(err)
						return
					}
					if stopRequested {
						_ = h.jobs.MarkPaused(ctx, jobID)
						return
					}
					scannedSinceCheckpoint = 0
				}
				continue
			}
			doc, ok, err := h.indexer.Build(ctx, a.AssetID)
			if err != nil {
				progress.Failed++
				progress.ErrorSamples = appendReindexError(progress.ErrorSamples, fmt.Sprintf("build %s: %v", a.AssetID, err))
				if scannedSinceCheckpoint >= reindexProgressCheckpointN {
					if err := h.jobs.UpdateProgress(ctx, jobID, progress); err != nil {
						failJob(err)
						return
					}
					stopRequested, err = h.jobs.IsStopRequested(ctx, jobID)
					if err != nil {
						failJob(err)
						return
					}
					if stopRequested {
						_ = h.jobs.MarkPaused(ctx, jobID)
						return
					}
					scannedSinceCheckpoint = 0
				}
				continue
			}
			if !ok {
				if job.DryRun {
					progress.DocumentsDeleted++
					if scannedSinceCheckpoint >= reindexProgressCheckpointN {
						if err := h.jobs.UpdateProgress(ctx, jobID, progress); err != nil {
							failJob(err)
							return
						}
						stopRequested, err = h.jobs.IsStopRequested(ctx, jobID)
						if err != nil {
							failJob(err)
							return
						}
						if stopRequested {
							_ = h.jobs.MarkPaused(ctx, jobID)
							return
						}
						scannedSinceCheckpoint = 0
					}
					continue
				}
				if err := h.es.DeleteDocument(ctx, a.AssetID); err != nil {
					progress.Failed++
					progress.ErrorSamples = appendReindexError(progress.ErrorSamples, fmt.Sprintf("delete %s: %v", a.AssetID, err))
					if scannedSinceCheckpoint >= reindexProgressCheckpointN {
						if err := h.jobs.UpdateProgress(ctx, jobID, progress); err != nil {
							failJob(err)
							return
						}
						stopRequested, err = h.jobs.IsStopRequested(ctx, jobID)
						if err != nil {
							failJob(err)
							return
						}
						if stopRequested {
							_ = h.jobs.MarkPaused(ctx, jobID)
							return
						}
						scannedSinceCheckpoint = 0
					}
					continue
				}
				progress.DocumentsDeleted++
				if scannedSinceCheckpoint >= reindexProgressCheckpointN {
					if err := h.jobs.UpdateProgress(ctx, jobID, progress); err != nil {
						failJob(err)
						return
					}
					stopRequested, err = h.jobs.IsStopRequested(ctx, jobID)
					if err != nil {
						failJob(err)
						return
					}
					if stopRequested {
						_ = h.jobs.MarkPaused(ctx, jobID)
						return
					}
					scannedSinceCheckpoint = 0
				}
				continue
			}
			if job.DryRun {
				progress.DocumentsIndexed++
				if scannedSinceCheckpoint >= reindexProgressCheckpointN {
					if err := h.jobs.UpdateProgress(ctx, jobID, progress); err != nil {
						failJob(err)
						return
					}
					stopRequested, err = h.jobs.IsStopRequested(ctx, jobID)
					if err != nil {
						failJob(err)
						return
					}
					if stopRequested {
						_ = h.jobs.MarkPaused(ctx, jobID)
						return
					}
					scannedSinceCheckpoint = 0
				}
				continue
			}
			docs = append(docs, espkg.BulkIndexDoc{ID: a.AssetID, Doc: doc})
			if scannedSinceCheckpoint >= reindexProgressCheckpointN {
				if err := h.jobs.UpdateProgress(ctx, jobID, progress); err != nil {
					failJob(err)
					return
				}
				stopRequested, err = h.jobs.IsStopRequested(ctx, jobID)
				if err != nil {
					failJob(err)
					return
				}
				if stopRequested {
					_ = h.jobs.MarkPaused(ctx, jobID)
					return
				}
				scannedSinceCheckpoint = 0
			}
		}

		if !job.DryRun && len(docs) > 0 {
			bulkResult, err := h.es.BulkIndex(ctx, docs)
			if err != nil {
				failJob(err)
				return
			}
			progress.DocumentsIndexed += int64(len(bulkResult.Succeeded))
			progress.Failed += int64(len(bulkResult.Failed))
			for _, item := range bulkResult.Failed {
				reason := item.Error
				if reason == "" {
					reason = fmt.Sprintf("status %d", item.Status)
				}
				progress.ErrorSamples = appendReindexError(progress.ErrorSamples, fmt.Sprintf("index %s: %s", item.ID, reason))
			}
		}

		progress.NextPage = page + 1
		if err := h.jobs.UpdateProgress(ctx, jobID, progress); err != nil {
			failJob(err)
			return
		}

		if int64(page*pageSize) >= total {
			break
		}
		page++
	}

	var esDocCount *int64
	if !job.DryRun {
		n, err := h.es.Count(ctx)
		if err == nil {
			esDocCount = &n
		}
	}
	_ = h.jobs.MarkSucceeded(ctx, jobID, esDocCount)
}
