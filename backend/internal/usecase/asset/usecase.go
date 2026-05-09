package asset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/id"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

var (
	ErrNotFound           = errors.New("asset not found")
	ErrInvalidRange       = errors.New("end_timestamp_ns must be greater than start_timestamp_ns")
	ErrMcapFileIDRequired = errors.New("mcap_file_id is required")
	ErrInvalidTag         = errors.New("invalid tag")
	ErrInvalidAssetID     = errors.New("asset id must be 8 alphanumeric characters")
	ErrAssetIDTaken       = errors.New("asset id already exists")
)

// defaultLifecycleMeta returns the lifecycle governance defaults for metadata.
// These fields are reserved at ingest time for future storage governance.
func defaultLifecycleMeta() map[string]interface{} {
	return map[string]interface{}{
		"retention_tier":     "standard",
		"archive_after_days": 90,
		"delete_after_days":  365,
		"total_size_bytes":   0,
		"last_accessed_at":   nil,
	}
}

type Usecase struct {
	repo           repository.AssetRepository
	tagRegistry    *config.TagRegistry
	algoRegistry   *config.AlgoRegistry
	tx             repository.TxRunner
	tagRepo        repository.AssetTagRepository
	algoLatestRepo repository.AssetAlgoLatestRepository
	eventRepo      repository.AssetEventRepository
}

func New(repo repository.AssetRepository) *Usecase {
	return &Usecase{repo: repo}
}

// NewWithTagRegistry creates a Usecase with tag validation support.
func NewWithTagRegistry(repo repository.AssetRepository, tagReg *config.TagRegistry) *Usecase {
	return &Usecase{repo: repo, tagRegistry: tagReg}
}

// NewFull creates a Usecase with tag validation and algo registry support.
// The algo registry is used to initialize algorithm states on asset creation.
func NewFull(repo repository.AssetRepository, tagReg *config.TagRegistry, algoReg *config.AlgoRegistry) *Usecase {
	return &Usecase{repo: repo, tagRegistry: tagReg, algoRegistry: algoReg}
}

// NewWithProjections wires the asset usecase with the projection and event
// repositories so asset mutations can update current-state tables and append
// business events in the same transaction.
func NewWithProjections(
	tx repository.TxRunner,
	repo repository.AssetRepository,
	tagRepo repository.AssetTagRepository,
	algoLatestRepo repository.AssetAlgoLatestRepository,
	eventRepo repository.AssetEventRepository,
	tagReg *config.TagRegistry,
	algoReg *config.AlgoRegistry,
) *Usecase {
	return &Usecase{
		repo:           repo,
		tagRegistry:    tagReg,
		algoRegistry:   algoReg,
		tx:             tx,
		tagRepo:        tagRepo,
		algoLatestRepo: algoLatestRepo,
		eventRepo:      eventRepo,
	}
}

func (u *Usecase) withMutationTx(ctx context.Context, fn func(context.Context) error) error {
	if u.tx == nil {
		return fn(ctx)
	}
	return u.tx.WithTx(ctx, fn)
}

func (u *Usecase) tagTypeFor(key string) string {
	if u.tagRegistry == nil {
		return "string"
	}
	if def, ok := u.tagRegistry.GetAllTags()[key]; ok && def.Type != "" {
		return def.Type
	}
	return "string"
}

func jsonPayload(v map[string]any) []byte {
	if v == nil {
		return []byte(`{}`)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{}`)
	}
	return b
}

func (u *Usecase) validateTags(tags map[string]string) error {
	if u.tagRegistry == nil {
		return nil
	}
	for k, v := range tags {
		if err := u.tagRegistry.Validate(k, v); err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidTag, err.Error())
		}
	}
	return nil
}

func (u *Usecase) appendAssetEvent(ctx context.Context, eventType string, a *models.Asset, payload map[string]any) error {
	if u.eventRepo == nil || a == nil {
		return nil
	}
	return u.eventRepo.Append(ctx, repository.AssetEventAppendInput{
		EventType:     eventType,
		AssetID:       a.AssetID,
		McapFileID:    a.McapFileID,
		TenantID:      a.TenantID,
		ProjectID:     a.ProjectID,
		EventSource:   "backend",
		RequestID:     middleware.RequestIDFromContext(ctx),
		EventPayload:  jsonPayload(payload),
		AggregateType: "asset",
	})
}

func (u *Usecase) upsertTagProjection(ctx context.Context, a *models.Asset, tags map[string]string, sourceType string) error {
	if u.tagRepo == nil || a == nil {
		return nil
	}
	for k, v := range tags {
		tagType := u.tagTypeFor(k)
		if err := u.tagRepo.Upsert(ctx, a.AssetID, k, v, tagType, sourceType); err != nil {
			return err
		}
		if err := u.appendAssetEvent(ctx, "tag_upserted", a, map[string]any{
			"tag_key":     k,
			"tag_value":   v,
			"tag_type":    tagType,
			"source_type": sourceType,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (u *Usecase) persistNewAsset(ctx context.Context, a *models.Asset, tags map[string]string) error {
	if err := u.withMutationTx(ctx, func(txCtx context.Context) error {
		if err := u.repo.InsertNew(txCtx, a); err != nil {
			return err
		}
		if err := u.seedInitialAlgoProjection(txCtx, a); err != nil {
			return err
		}
		if err := u.appendAssetEvent(txCtx, "asset_created", a, map[string]any{
			"asset_id":        a.AssetID,
			"mcap_file_id":    a.McapFileID,
			"segment_locator": a.SegmentLocator,
			"lifecycle_state": a.LifecycleState,
			"asset_type":      a.AssetType,
			"owner":           a.Owner,
			"reviewer":        a.Reviewer,
		}); err != nil {
			return err
		}
		return u.upsertTagProjection(txCtx, a, tags, "manual")
	}); err != nil {
		return err
	}
	// Secondary index write is best-effort for now.
	return u.repo.WriteSegmentIndex(ctx, a)
}

const maxAssetIDAllocationAttempts = 32

// allocateNewAssetID assigns a random 8-char asset_id and persists the new asset, retrying on id collision.
func (u *Usecase) allocateNewAssetID(ctx context.Context, a *models.Asset, tags map[string]string) error {
	for range maxAssetIDAllocationAttempts {
		gid, err := id.GenerateAssetID()
		if err != nil {
			return err
		}
		a.AssetID = gid
		a.Version = 0
		a.CreatedAt = time.Time{}
		if err := u.persistNewAsset(ctx, a, tags); err != nil {
			if errors.Is(err, repository.ErrDuplicateAssetID) {
				continue
			}
			return err
		}
		return nil
	}
	return fmt.Errorf("exhausted asset id allocation attempts (%d)", maxAssetIDAllocationAttempts)
}

func parseAlgoStatusKey(key string) (algoName, algoVersion string, ok bool) {
	const suffix = ":" + models.AlgoFieldStatus
	if !strings.HasSuffix(key, suffix) {
		return "", "", false
	}
	algoKey := strings.TrimSuffix(key, suffix)
	at := strings.LastIndex(algoKey, "@")
	if at <= 0 || at >= len(algoKey)-1 {
		return "", "", false
	}
	return algoKey[:at], algoKey[at+1:], true
}

func (u *Usecase) seedInitialAlgoProjection(ctx context.Context, a *models.Asset) error {
	if u.algoLatestRepo == nil || a == nil || len(a.AlgoResults) == 0 {
		return nil
	}
	for key, status := range a.AlgoResults {
		algoName, algoVersion, ok := parseAlgoStatusKey(key)
		if !ok {
			continue
		}
		if err := u.algoLatestRepo.Upsert(ctx, &models.AssetAlgoLatest{
			AssetID:     a.AssetID,
			AlgoName:    algoName,
			AlgoVersion: algoVersion,
			Status:      status,
			TenantID:    a.TenantID,
			ProjectID:   a.ProjectID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (u *Usecase) hydrateTags(ctx context.Context, a *models.Asset) error {
	if a == nil || u.tagRepo == nil {
		return nil
	}
	rows, err := u.tagRepo.ListByAsset(ctx, a.AssetID)
	if err != nil {
		return err
	}
	a.Tags = map[string]string{}
	for _, row := range rows {
		a.Tags[row.TagKey] = row.TagValue
	}
	return nil
}

func timeStringPtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func (u *Usecase) hydrateAlgoResults(ctx context.Context, a *models.Asset) error {
	if a == nil || u.algoLatestRepo == nil {
		return nil
	}
	rows, err := u.algoLatestRepo.ListByAsset(ctx, a.AssetID)
	if err != nil {
		return err
	}
	a.AlgoResults = map[string]string{}
	for _, row := range rows {
		algoKey := row.AlgoName + "@" + row.AlgoVersion
		if row.Status != "" {
			a.AlgoResults[algoKey+":"+models.AlgoFieldStatus] = row.Status
		}
		if row.RunID != "" {
			a.AlgoResults[algoKey+":"+models.AlgoFieldRunID] = row.RunID
		}
		if row.Method != "" {
			a.AlgoResults[algoKey+":"+models.AlgoFieldMethod] = row.Method
		}
		if row.OutputURI != "" {
			a.AlgoResults[algoKey+":"+models.AlgoFieldOutputURI] = row.OutputURI
		}
		if startedAt := timeStringPtr(row.StartedAt); startedAt != "" {
			a.AlgoResults[algoKey+":"+models.AlgoFieldStartedAt] = startedAt
		}
		if finishedAt := timeStringPtr(row.FinishedAt); finishedAt != "" {
			a.AlgoResults[algoKey+":"+models.AlgoFieldFinishedAt] = finishedAt
		}
		if row.ErrorMessage != "" {
			a.AlgoResults[algoKey+":"+models.AlgoFieldReason] = row.ErrorMessage
		}
	}
	return nil
}

func (u *Usecase) hydrateAssetReadModels(ctx context.Context, a *models.Asset) error {
	if a == nil {
		return nil
	}
	if err := u.hydrateTags(ctx, a); err != nil {
		return err
	}
	if err := u.hydrateAlgoResults(ctx, a); err != nil {
		return err
	}
	return nil
}

func (u *Usecase) hydrateAssetsReadModels(ctx context.Context, items []*models.Asset) error {
	for _, a := range items {
		if err := u.hydrateAssetReadModels(ctx, a); err != nil {
			return err
		}
	}
	return nil
}

type CreateInput struct {
	AssetID             string
	McapFileID          string
	StartTimestampNs    int64
	EndTimestampNs      int64
	Reviewer            string
	Owner               string
	SegType             string
	AssetType           string
	Status              string
	LifecycleState      string
	Env                 string
	Task                string
	Tags                map[string]string
	Files               map[string]string
	Metadata            map[string]interface{}
	LifecycleMeta       map[string]interface{}
	RetentionTier       string
	ExpireAt            *time.Time
	StorageURI          string
	ThumbURI            string
	AssetLevel          int
	ParentAssetID       string
	RootAssetID         string
	SegmentIndex        *int
	ParentStartOffsetMs *int64
	ParentEndOffsetMs   *int64
	SplitMethod         string
	SplitAlgoName       string
	SplitAlgoVersion    string
	SplitRunID          string
	SplitReason         string
	DeliveryCount       int
	LastDeliveredAt     *time.Time
	LastDeliveredTo     string
}

type UpdateInput struct {
	Status   *string
	Reviewer *string
	Owner    *string
	Tags     map[string]string
}

type ListEventsInput struct {
	EventTypes        []string
	EventTypePatterns []string
	AlgoKey           string
	BeforeEventSeq    *int64
	AfterEventSeq     *int64
	StartTime         *time.Time
	EndTime           *time.Time
	Limit             int
}

type ListEventsResult struct {
	Items      []*models.AssetEvent
	NextCursor *int64
}

type UpsertTagInput struct {
	Key   string
	Value string
}

type CommitSegmentsInput struct {
	McapFileID string
	Ranges     [][2]int64
	Reviewer   string
	Owner      string
}

func (u *Usecase) Get(ctx context.Context, assetID string) (*models.Asset, error) {
	a, err := u.repo.Get(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrNotFound
	}
	if err := u.hydrateAssetReadModels(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

// BatchGet returns multiple assets by their IDs, skipping not-found ones.
func (u *Usecase) BatchGet(ctx context.Context, assetIDs []string) ([]*models.Asset, error) {
	items := make([]*models.Asset, 0, len(assetIDs))
	for _, id := range assetIDs {
		a, err := u.Get(ctx, id)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, err
		}
		items = append(items, a)
	}
	return items, nil
}

func (u *Usecase) ListByMcapFile(ctx context.Context, mcapFileID string) ([]*models.Asset, error) {
	if mcapFileID == "" {
		return nil, ErrMcapFileIDRequired
	}
	items, err := u.repo.ListByMcapFile(ctx, mcapFileID)
	if err != nil {
		return nil, err
	}
	if err := u.hydrateAssetsReadModels(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

// ListWithFilters queries assets using a parameterized WHERE clause with pagination and sorting.
func (u *Usecase) ListWithFilters(ctx context.Context, whereSQL string, args []interface{},
	page, pageSize int, orderBy string) ([]*models.Asset, int64, error) {
	items, total, err := u.repo.ListWithFilters(ctx, whereSQL, args, page, pageSize, orderBy)
	if err != nil {
		return nil, 0, err
	}
	if err := u.hydrateAssetsReadModels(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (u *Usecase) ListEvents(ctx context.Context, assetID string, in ListEventsInput) (*ListEventsResult, error) {
	a, err := u.repo.Get(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrNotFound
	}
	if u.eventRepo == nil {
		return &ListEventsResult{Items: []*models.AssetEvent{}}, nil
	}

	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	queryLimit := limit + 1
	rows, err := u.eventRepo.ListByAsset(ctx, assetID, repository.AssetEventListOptions{
		EventTypes:        in.EventTypes,
		EventTypePatterns: in.EventTypePatterns,
		AlgoKey:           in.AlgoKey,
		BeforeEventSeq:    in.BeforeEventSeq,
		AfterEventSeq:     in.AfterEventSeq,
		StartTime:         in.StartTime,
		EndTime:           in.EndTime,
		Limit:             queryLimit,
	})
	if err != nil {
		return nil, err
	}

	res := &ListEventsResult{Items: rows}
	if len(rows) > limit {
		rows = rows[:limit]
		res.Items = rows
		cursor := rows[len(rows)-1].EventSeq
		res.NextCursor = &cursor
	}
	if res.Items == nil {
		res.Items = []*models.AssetEvent{}
	}
	return res, nil
}

func (u *Usecase) ListTagHistory(ctx context.Context, assetID string, in ListEventsInput) (*ListEventsResult, error) {
	in.EventTypes = []string{"tag_upserted", "tag_deleted"}
	in.EventTypePatterns = nil
	in.AlgoKey = ""
	return u.ListEvents(ctx, assetID, in)
}

func (u *Usecase) Create(ctx context.Context, in CreateInput) (*models.Asset, error) {
	if in.McapFileID == "" {
		return nil, ErrMcapFileIDRequired
	}
	if in.EndTimestampNs <= in.StartTimestampNs {
		return nil, ErrInvalidRange
	}
	if in.ParentAssetID != "" && !id.ValidateAssetID(in.ParentAssetID) {
		return nil, ErrInvalidAssetID
	}
	if in.RootAssetID != "" && !id.ValidateAssetID(in.RootAssetID) {
		return nil, ErrInvalidAssetID
	}
	if in.AssetID != "" && !id.ValidateAssetID(in.AssetID) {
		return nil, ErrInvalidAssetID
	}
	tags := in.Tags
	if tags == nil {
		tags = map[string]string{}
	}
	if err := u.validateTags(tags); err != nil {
		return nil, err
	}
	a := &models.Asset{
		AssetID:             "",
		McapFileID:          in.McapFileID,
		StartTimestampNs:    in.StartTimestampNs,
		EndTimestampNs:      in.EndTimestampNs,
		DurationSec:         float64(in.EndTimestampNs-in.StartTimestampNs) / 1e9,
		Reviewer:            in.Reviewer,
		Status:              models.AssetStatusApproved,
		Owner:               in.Owner,
		SegType:             in.SegType,
		AssetType:           in.AssetType,
		LifecycleState:      in.LifecycleState,
		Env:                 in.Env,
		Task:                in.Task,
		Tags:                tags,
		AlgoResults:         map[string]string{},
		Files:               map[string]string{},
		Metadata:            map[string]interface{}{},
		LifecycleMeta:       defaultLifecycleMeta(),
		RetentionTier:       in.RetentionTier,
		ExpireAt:            in.ExpireAt,
		StorageURI:          in.StorageURI,
		ThumbURI:            in.ThumbURI,
		AssetLevel:          in.AssetLevel,
		ParentAssetID:       in.ParentAssetID,
		RootAssetID:         in.RootAssetID,
		SegmentIndex:        in.SegmentIndex,
		ParentStartOffsetMs: in.ParentStartOffsetMs,
		ParentEndOffsetMs:   in.ParentEndOffsetMs,
		SplitMethod:         in.SplitMethod,
		SplitAlgoName:       in.SplitAlgoName,
		SplitAlgoVersion:    in.SplitAlgoVersion,
		SplitRunID:          in.SplitRunID,
		SplitReason:         in.SplitReason,
		DeliveryCount:       in.DeliveryCount,
		LastDeliveredAt:     in.LastDeliveredAt,
		LastDeliveredTo:     in.LastDeliveredTo,
		CreatedAt:           time.Now(),
	}
	if in.AssetID != "" {
		a.AssetID = in.AssetID
	}
	if in.Status != "" {
		a.Status = models.AssetStatus(in.Status)
	}
	if a.AssetType == "" {
		a.AssetType = in.SegType
	}
	if a.SegType == "" {
		a.SegType = a.AssetType
	}
	for k, v := range in.Files {
		a.Files[k] = v
	}
	if len(in.Metadata) > 0 {
		a.Metadata = in.Metadata
	}
	if len(in.LifecycleMeta) > 0 {
		a.LifecycleMeta = in.LifecycleMeta
	}
	if a.RetentionTier == "" {
		if v, ok := a.LifecycleMeta["retention_tier"].(string); ok && v != "" {
			a.RetentionTier = v
		}
	}
	if a.StorageURI == "" {
		if raw, ok := a.Files["raw_mcap"]; ok {
			a.StorageURI = raw
		}
	}
	if a.ThumbURI == "" {
		if thumb, ok := a.Files["thumbnail"]; ok {
			a.ThumbURI = thumb
		}
	}
	// Initialize algorithm states from algo_registry if available.
	if u.algoRegistry != nil {
		initAlgoStates(a, u.algoRegistry)
		// Write raw_mcap reference to files.
		if _, ok := a.Files["raw_mcap"]; !ok {
			a.Files["raw_mcap"] = in.McapFileID
		}
	}
	if in.AssetID == "" {
		if err := u.allocateNewAssetID(ctx, a, tags); err != nil {
			return nil, err
		}
		return a, nil
	}
	if err := u.persistNewAsset(ctx, a, tags); err != nil {
		if errors.Is(err, repository.ErrDuplicateAssetID) {
			return nil, ErrAssetIDTaken
		}
		return nil, err
	}
	return a, nil
}

func (u *Usecase) Update(ctx context.Context, assetID string, in UpdateInput) (*models.Asset, error) {
	a, err := u.repo.Get(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrNotFound
	}
	prevStatus := string(a.Status)
	prevLifecycle := a.LifecycleState
	if in.Status != nil {
		a.Status = models.AssetStatus(*in.Status)
	}
	if in.Reviewer != nil {
		a.Reviewer = *in.Reviewer
	}
	if in.Owner != nil {
		a.Owner = *in.Owner
	}
	if err := u.validateTags(in.Tags); err != nil {
		return nil, err
	}
	for k, v := range in.Tags {
		if a.Tags == nil {
			a.Tags = map[string]string{}
		}
		a.Tags[k] = v
	}

	if err := u.withMutationTx(ctx, func(txCtx context.Context) error {
		if err := u.repo.Set(txCtx, a); err != nil {
			return err
		}
		if err := u.appendAssetEvent(txCtx, "asset_updated", a, map[string]any{
			"asset_id":        a.AssetID,
			"lifecycle_state": a.LifecycleState,
			"owner":           a.Owner,
			"reviewer":        a.Reviewer,
		}); err != nil {
			return err
		}
		if in.Status != nil && (prevStatus != string(a.Status) || prevLifecycle != a.LifecycleState) {
			if err := u.appendAssetEvent(txCtx, "asset_lifecycle_changed", a, map[string]any{
				"prev_status":          prevStatus,
				"new_status":           string(a.Status),
				"prev_lifecycle_state": prevLifecycle,
				"new_lifecycle_state":  a.LifecycleState,
			}); err != nil {
				return err
			}
		}
		return u.upsertTagProjection(txCtx, a, in.Tags, "manual")
	}); err != nil {
		return nil, err
	}
	return a, nil
}

func (u *Usecase) UpsertTag(ctx context.Context, assetID string, in UpsertTagInput) (*models.Asset, error) {
	if err := u.validateTags(map[string]string{in.Key: in.Value}); err != nil {
		return nil, err
	}

	a, err := u.repo.Get(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrNotFound
	}

	if err := u.withMutationTx(ctx, func(txCtx context.Context) error {
		return u.upsertTagProjection(txCtx, a, map[string]string{in.Key: in.Value}, "manual")
	}); err != nil {
		return nil, err
	}
	return u.Get(ctx, assetID)
}

func (u *Usecase) findTag(ctx context.Context, assetID, tagKey string) (*models.AssetTag, error) {
	if u.tagRepo == nil {
		return nil, nil
	}
	rows, err := u.tagRepo.ListByAsset(ctx, assetID)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.TagKey == tagKey {
			return row, nil
		}
	}
	return nil, nil
}

func (u *Usecase) DeleteTag(ctx context.Context, assetID, tagKey string) (*models.Asset, error) {
	a, err := u.repo.Get(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrNotFound
	}

	if err := u.withMutationTx(ctx, func(txCtx context.Context) error {
		existing, err := u.findTag(txCtx, assetID, tagKey)
		if err != nil {
			return err
		}
		if existing == nil {
			return nil
		}
		if err := u.tagRepo.Delete(txCtx, assetID, tagKey); err != nil {
			return err
		}
		return u.appendAssetEvent(txCtx, "tag_deleted", a, map[string]any{
			"tag_key":     existing.TagKey,
			"tag_value":   existing.TagValue,
			"tag_type":    existing.TagType,
			"source_type": "manual",
		})
	}); err != nil {
		return nil, err
	}
	return u.Get(ctx, assetID)
}

func (u *Usecase) Delete(ctx context.Context, assetID string) error {
	a, err := u.repo.Get(ctx, assetID)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrNotFound
	}
	prevStatus := string(a.Status)
	prevLifecycle := a.LifecycleState
	return u.withMutationTx(ctx, func(txCtx context.Context) error {
		if err := u.repo.SoftDelete(txCtx, assetID); err != nil {
			return err
		}
		return u.appendAssetEvent(txCtx, "asset_lifecycle_changed", a, map[string]any{
			"prev_status":          prevStatus,
			"new_status":           string(models.AssetStatusArchived),
			"prev_lifecycle_state": prevLifecycle,
			"new_lifecycle_state":  "archived",
		})
	})
}

func (u *Usecase) CommitSegments(ctx context.Context, in CommitSegmentsInput) ([]string, error) {
	if in.McapFileID == "" {
		return nil, ErrMcapFileIDRequired
	}
	var created []string
	for _, r := range in.Ranges {
		startNs, endNs := r[0], r[1]
		if endNs <= startNs {
			return created, ErrInvalidRange
		}
		a := &models.Asset{
			McapFileID:       in.McapFileID,
			StartTimestampNs: startNs,
			EndTimestampNs:   endNs,
			DurationSec:      float64(endNs-startNs) / 1e9,
			Reviewer:         in.Reviewer,
			Status:           models.AssetStatusApproved,
			Owner:            in.Owner,
			AlgoResults:      map[string]string{},
			Tags:             map[string]string{},
			Files:            map[string]string{},
			LifecycleMeta:    defaultLifecycleMeta(),
			CreatedAt:        time.Now(),
		}
		// Initialize algorithm states from algo_registry if available.
		if u.algoRegistry != nil {
			initAlgoStates(a, u.algoRegistry)
			a.Files["raw_mcap"] = in.McapFileID
		}
		if err := u.allocateNewAssetID(ctx, a, a.Tags); err != nil {
			return created, err
		}
		created = append(created, a.AssetID)
	}
	return created, nil
}

// initAlgoStates initializes algorithm statuses on a new asset based on the algo registry.
// Algorithms with no dependencies get status "pending"; those with dependencies get "blocked".
func initAlgoStates(a *models.Asset, reg *config.AlgoRegistry) {
	allAlgos := reg.GetAllAlgorithms()
	for algoName, def := range allAlgos {
		for _, ver := range def.Versions {
			algoKey := algoName + "@" + ver
			statusKey := algoKey + ":" + models.AlgoFieldStatus
			if len(def.DependsOn) == 0 {
				a.AlgoResults[statusKey] = string(models.AlgoStatusPending)
			} else {
				a.AlgoResults[statusKey] = string(models.AlgoStatusBlocked)
			}
		}
	}
}
