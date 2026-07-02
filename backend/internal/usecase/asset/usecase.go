package asset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CyberOrigin2077/cyber-databrew/internal/config"
	"github.com/CyberOrigin2077/cyber-databrew/internal/deliveryrules"
	"github.com/CyberOrigin2077/cyber-databrew/internal/filter"
	"github.com/CyberOrigin2077/cyber-databrew/internal/id"
	"github.com/CyberOrigin2077/cyber-databrew/internal/middleware"
	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound           = errors.New("asset not found")
	ErrInvalidRange       = errors.New("end_timestamp_ns must be greater than start_timestamp_ns")
	ErrMcapFileIDRequired = errors.New("mcap_file_id is required")
	ErrInvalidTag         = errors.New("invalid tag")
	ErrTagSourceInvalid   = errors.New("invalid tag source")
	ErrTagImmutable       = errors.New("tag source is immutable")
	ErrInvalidAssetID     = errors.New("asset id must be 8 alphanumeric characters")
	ErrAssetIDTaken       = errors.New("asset id already exists")
	ErrInvalidMcapFileID  = errors.New("mcap_file_id must be exactly 8 alphanumeric characters")
	ErrMcapFileNotFound   = errors.New("mcap_file_id does not exist")
	ErrCustomerNotFound   = errors.New("customer not found for customer.* tag namespace") // CYB-1070
)

func mapCreateDBError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	if pgErr.Code == "23514" && pgErr.ConstraintName == "assets_mcap_file_id_check" {
		return ErrInvalidMcapFileID
	}
	if pgErr.Code == "23503" && pgErr.ConstraintName == "fk_assets_mcap" {
		return ErrMcapFileNotFound
	}
	return err
}

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
	logicalRepo    repository.LogicalAssetRepository
	tagRegistry    *config.TagRegistry
	algoRegistry   *config.AlgoRegistry
	tx             repository.TxRunner
	tagRepo        repository.AssetTagRepository
	algoLatestRepo repository.AssetAlgoLatestRepository
	eventRepo      repository.AssetEventRepository
	customerRepo   repository.CustomerRepository       // CYB-1070: customer.* namespace lint
	usageStatsRepo repository.AssetUsageStatRepository // CYB-1095/1096: usage stats
	validator      *deliveryrules.AssetWriteValidator  // CYB-1164: hierarchy invariants
	schemaRegistry *models.SchemaRegistry
}

func New(repo repository.AssetRepository) *Usecase {
	return &Usecase{repo: repo, schemaRegistry: models.NewSchemaRegistry()}
}

// NewWithTagRegistry creates a Usecase with tag validation support.
func NewWithTagRegistry(repo repository.AssetRepository, tagReg *config.TagRegistry) *Usecase {
	return &Usecase{repo: repo, tagRegistry: tagReg, schemaRegistry: models.NewSchemaRegistry()}
}

// NewFull creates a Usecase with tag validation and algo registry support.
// The algo registry is used to initialize algorithm states on asset creation.
func NewFull(repo repository.AssetRepository, tagReg *config.TagRegistry, algoReg *config.AlgoRegistry) *Usecase {
	return &Usecase{repo: repo, tagRegistry: tagReg, algoRegistry: algoReg, schemaRegistry: models.NewSchemaRegistry()}
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
		schemaRegistry: models.NewSchemaRegistry(),
	}
}

// SetLogicalAssetRepo wires logical_assets persistence (CYB-1013).
func (u *Usecase) SetLogicalAssetRepo(r repository.LogicalAssetRepository) {
	u.logicalRepo = r
}

// SetCustomerRepo wires customer persistence for customer.* namespace lint (CYB-1070).
func (u *Usecase) SetCustomerRepo(r repository.CustomerRepository) {
	u.customerRepo = r
}

// SetUsageStatsRepo wires usage stats persistence for view/favorite counters (CYB-1095/1096).
func (u *Usecase) SetUsageStatsRepo(r repository.AssetUsageStatRepository) {
	u.usageStatsRepo = r
}

// SetValidator wires the asset hierarchy validator (CYB-1164).
func (u *Usecase) SetValidator(v *deliveryrules.AssetWriteValidator) {
	u.validator = v
}

func (u *Usecase) SetSchemaRegistry(r *models.SchemaRegistry) {
	u.schemaRegistry = r
}

func (u *Usecase) GetAssetTypeSchema(assetType string) (json.RawMessage, bool) {
	if u.schemaRegistry == nil {
		return nil, false
	}
	schema := u.schemaRegistry.GetSchema(assetType)
	if schema == nil {
		return nil, false
	}
	return schema, true
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

// validateCustomerNamespace checks that tags with key prefix "customer." reference
// an existing customer (CYB-1070). The customer ID is extracted from the tag
// value (e.g. key="customer.id", value="cust_abc").
func (u *Usecase) validateCustomerNamespace(ctx context.Context, tags map[string]string) error {
	if u.customerRepo == nil {
		return nil
	}
	for k, v := range tags {
		if !strings.HasPrefix(k, "customer.") {
			continue
		}
		// For customer.id tags the value IS the customer ID.
		// For other customer.* tags, the value may also be a customer ID.
		exists, err := u.customerRepo.Exists(ctx, v)
		if err != nil {
			return fmt.Errorf("customer namespace check: %w", err)
		}
		if !exists {
			return fmt.Errorf("%w: customer %q not found for tag %q", ErrCustomerNotFound, v, k)
		}
	}
	return nil
}

// propagateTagToDescendants applies a tag to all descendant assets when the
// tag registry declares propagation=descendants (CYB-1068).
func (u *Usecase) propagateTagToDescendants(ctx context.Context, assetID, tagKey, tagValue, tagType string, src tagSource) error {
	if u.tagRegistry == nil || !u.tagRegistry.ShouldPropagate(tagKey) {
		return nil
	}
	descendants, err := u.repo.ListDescendants(ctx, assetID)
	if err != nil {
		return fmt.Errorf("tag propagation: %w", err)
	}
	for _, desc := range descendants {
		if err := u.tagRepo.Upsert(ctx, repository.AssetTagUpsertInput{
			AssetID:       desc.AssetID,
			TagKey:        tagKey,
			TagValue:      tagValue,
			TagType:       tagType,
			SourceType:    src.SourceType,
			SourceName:    src.SourceName,
			SourceVersion: src.SourceVersion,
			RunID:         src.RunID,
			TenantID:      desc.TenantID,
			ProjectID:     desc.ProjectID,
		}); err != nil {
			return fmt.Errorf("tag propagation to %s: %w", desc.AssetID, err)
		}
	}
	return nil
}

// validateTagSource enforces tag_registry tag_sources[] identity contracts
// for a single tag write (requires_source_name / requires_source_version).
// Unknown sources return ErrTagSourceInvalid only when the registry has any
// tag_sources configured (back-compat for environments without the block).
func (u *Usecase) validateTagSource(sourceType, sourceName, sourceVersion string) error {
	if u.tagRegistry == nil {
		return nil
	}
	if err := u.tagRegistry.ValidateSource(sourceType, sourceName, sourceVersion); err != nil {
		return fmt.Errorf("%w: %s", ErrTagSourceInvalid, err.Error())
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

// tagSource captures the per-write identity of a tag assertion. Empty fields
// other than SourceType are allowed when the registry does not require them.
type tagSource struct {
	SourceType    string
	SourceName    string
	SourceVersion string
	RunID         string
}

func defaultTagSource() tagSource { return tagSource{SourceType: "system"} }

func (u *Usecase) upsertTagProjection(ctx context.Context, a *models.Asset, tags map[string]string, src tagSource) error {
	if u.tagRepo == nil || a == nil {
		return nil
	}
	for k, v := range tags {
		tagType := u.tagTypeFor(k)
		if err := u.assertNotImmutable(ctx, a.AssetID, k, src); err != nil {
			return err
		}
		if err := u.tagRepo.Upsert(ctx, repository.AssetTagUpsertInput{
			AssetID:       a.AssetID,
			TagKey:        k,
			TagValue:      v,
			TagType:       tagType,
			SourceType:    src.SourceType,
			SourceName:    src.SourceName,
			SourceVersion: src.SourceVersion,
			RunID:         src.RunID,
			TenantID:      a.TenantID,
			ProjectID:     a.ProjectID,
		}); err != nil {
			return err
		}
		if err := u.appendAssetEvent(ctx, "tag_upserted", a, map[string]any{
			"tag_key":        k,
			"tag_value":      v,
			"tag_type":       tagType,
			"source_type":    src.SourceType,
			"source_name":    src.SourceName,
			"source_version": src.SourceVersion,
			"run_id":         src.RunID,
		}); err != nil {
			return err
		}
		// CYB-1068: propagate to descendants when tag declares propagation=descendants.
		if err := u.propagateTagToDescendants(ctx, a.AssetID, k, v, tagType, src); err != nil {
			return err
		}
	}
	return nil
}

// assertNotImmutable rejects any attempt to rewrite an existing assertion
// from a source flagged `immutable: true` in tag_registry.yaml.
func (u *Usecase) assertNotImmutable(ctx context.Context, assetID, tagKey string, src tagSource) error {
	if u.tagRegistry == nil || u.tagRepo == nil {
		return nil
	}
	def, ok := u.tagRegistry.SourceDef(src.SourceType)
	if !ok || !def.Immutable {
		return nil
	}
	existing, err := u.tagRepo.ListByAsset(ctx, assetID)
	if err != nil {
		return err
	}
	for _, row := range existing {
		if row.TagKey == tagKey && row.SourceType == src.SourceType &&
			row.SourceVersion == src.SourceVersion {
			return fmt.Errorf("%w: %s/%s on %s", ErrTagImmutable, src.SourceType, tagKey, assetID)
		}
	}
	return nil
}

func (u *Usecase) persistNewAsset(ctx context.Context, a *models.Asset, tags map[string]string, promoteLogicalID string) error {
	if err := u.withMutationTx(ctx, func(txCtx context.Context) error {
		if promoteLogicalID != "" {
			promoteIn, err := u.preparePromoteVersion(txCtx, a, promoteLogicalID)
			if err != nil {
				return err
			}
			if err := u.finalizePromoteVersion(txCtx, a, promoteIn); err != nil {
				return err
			}
		} else {
			if err := u.seedFirstVersion(txCtx, a); err != nil {
				return err
			}
			if err := u.repo.InsertNew(txCtx, a); err != nil {
				return err
			}
			if err := u.appendAssetEvent(txCtx, "asset_created", a, map[string]any{
				"asset_id":         a.AssetID,
				"mcap_file_id":     a.McapFileID,
				"segment_locator":  a.SegmentLocator,
				"lifecycle_state":  a.LifecycleState,
				"asset_type":       a.AssetType,
				"logical_asset_id": a.LogicalAssetID,
				"revision":         a.Revision,
				"owner":            a.Owner,
				"reviewer":         a.Reviewer,
			}); err != nil {
				return err
			}
		}
		if err := u.seedInitialAlgoProjection(txCtx, a); err != nil {
			return err
		}
		return u.upsertTagProjection(txCtx, a, tags, defaultTagSource())
	}); err != nil {
		return err
	}
	return u.repo.WriteSegmentIndex(ctx, a)
}

const maxAssetIDAllocationAttempts = 32

// allocateNewAssetID assigns a random 8-char asset_id and persists the new asset, retrying on id collision.
func (u *Usecase) allocateNewAssetID(ctx context.Context, a *models.Asset, tags map[string]string, promoteLogicalID string) error {
	for range maxAssetIDAllocationAttempts {
		gid, err := id.GenerateAssetID()
		if err != nil {
			return err
		}
		a.AssetID = gid
		a.Version = 0
		a.CreatedAt = time.Time{}
		if err := u.persistNewAsset(ctx, a, tags, promoteLogicalID); err != nil {
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
	// Flat map is last-applied-wins per key for backward compatibility.
	// tags_detailed is the source of truth for multi-source assertions.
	a.Tags = map[string]string{}
	latestApplied := map[string]time.Time{}
	detailed := make([]models.AssetTag, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		detailed = append(detailed, *row)
		if t, seen := latestApplied[row.TagKey]; !seen || row.AppliedAt.After(t) {
			a.Tags[row.TagKey] = row.TagValue
			latestApplied[row.TagKey] = row.AppliedAt
		}
	}
	a.TagsDetailed = detailed
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
	LogicalAssetID      string
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
	Status         *string
	LifecycleState *string
	Reviewer       *string
	Owner          *string
	Tags           map[string]string
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
	// Source identity (CYB-1015). When SourceType is empty the handler must
	// default it (typically to "human" on UI flows, "system" elsewhere).
	SourceType    string
	SourceName    string
	SourceVersion string
	RunID         string
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

// GetAll returns an asset by ID regardless of soft-delete status.
// Used by GET /assets/:id so the API can return archived assets.
func (u *Usecase) GetAll(ctx context.Context, assetID string) (*models.Asset, error) {
	a, err := u.repo.GetAll(ctx, assetID)
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
	page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, int64, error) {
	items, total, err := u.repo.ListWithFilters(ctx, whereSQL, args, page, pageSize, orderBy)
	if err != nil {
		return nil, 0, err
	}
	if err := u.hydrateAssetsReadModels(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

type assetListPageRepo interface {
	ListWithFiltersPage(ctx context.Context, whereSQL string, args []interface{}, page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, error)
}

// ListWithFiltersPage returns one page without COUNT(*). Falls back to ListWithFilters when unsupported.
func (u *Usecase) ListWithFiltersPage(ctx context.Context, whereSQL string, args []interface{},
	page, pageSize int, orderBy filter.OrderByClause) ([]*models.Asset, error) {
	var (
		items []*models.Asset
		err   error
	)
	if pageRepo, ok := u.repo.(assetListPageRepo); ok {
		items, err = pageRepo.ListWithFiltersPage(ctx, whereSQL, args, page, pageSize, orderBy)
	} else {
		items, _, err = u.repo.ListWithFilters(ctx, whereSQL, args, page, pageSize, orderBy)
	}
	if err != nil {
		return nil, err
	}
	if err := u.hydrateAssetsReadModels(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
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

// ListGlobalEvents returns recent events across all assets. Does not require
// asset existence check. Falls back to the last 24 hours when no StartTime
// is provided (prevents full table scans on asset_events).
func (u *Usecase) ListGlobalEvents(ctx context.Context, in ListEventsInput) (*ListEventsResult, error) {
	if u.eventRepo == nil {
		return &ListEventsResult{Items: []*models.AssetEvent{}}, nil
	}

	limit := in.Limit
	if limit <= 0 {
		limit = 100
	}
	queryLimit := limit + 1
	rows, err := u.eventRepo.ListGlobal(ctx, repository.AssetEventListOptions{
		EventTypes:        in.EventTypes,
		EventTypePatterns: in.EventTypePatterns,
		BeforeEventSeq:    in.BeforeEventSeq,
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

func (u *Usecase) Create(ctx context.Context, in CreateInput) (*models.Asset, error) {
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
	promoteLogicalID := strings.TrimSpace(in.LogicalAssetID)
	if promoteLogicalID != "" && !id.ValidateAssetID(promoteLogicalID) {
		return nil, ErrInvalidAssetID
	}
	tags := in.Tags
	if tags == nil {
		tags = map[string]string{}
	}
	if err := u.validateTags(tags); err != nil {
		return nil, err
	}
	// CYB-1070: customer.* namespace lint.
	if err := u.validateCustomerNamespace(ctx, tags); err != nil {
		return nil, err
	}
	assetType := strings.TrimSpace(in.AssetType)
	segType := strings.TrimSpace(in.SegType)
	if assetType == "" {
		assetType = segType
	}
	if segType == "" {
		segType = assetType
	}
	// Keep create behavior stable for legacy callers that omit both fields.
	if assetType == "" {
		assetType = "segment"
		segType = "segment"
	}
	mcapFileID := in.McapFileID
	if mcapFileID == "" && assetType != "derived_asset" && (u.schemaRegistry == nil || u.schemaRegistry.GetSchema(assetType) == nil) {
		return nil, ErrMcapFileIDRequired
	}
	// Schema-registered types (e.g. grace_video) may omit mcap_file_id; the
	// DB column allows empty/NULL per assets_mcap_file_id_check.
	a := &models.Asset{
		AssetID:             "",
		McapFileID:          mcapFileID,
		StartTimestampNs:    in.StartTimestampNs,
		EndTimestampNs:      in.EndTimestampNs,
		DurationSec:         float64(in.EndTimestampNs-in.StartTimestampNs) / 1e9,
		Reviewer:            in.Reviewer,
		Status:              models.AssetStatusApproved,
		Owner:               in.Owner,
		SegType:             segType,
		AssetType:           assetType,
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
	for k, v := range in.Files {
		a.Files[k] = v
	}
	if len(in.Metadata) > 0 {
		a.Metadata = in.Metadata
	}
	if u.schemaRegistry != nil {
		if err := u.schemaRegistry.Validate(a.AssetType, a.Metadata); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidTag, err.Error())
		}
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
		// Write raw_mcap reference to files for non-schema types.
		if _, ok := a.Files["raw_mcap"]; !ok && mcapFileID != "" {
			a.Files["raw_mcap"] = mcapFileID
		}
	}
	// CYB-1164: validate hierarchy invariants before persisting.
	if u.validator != nil {
		if err := u.validator.ValidateCreate(ctx, a); err != nil {
			return nil, err
		}
	}
	if in.AssetID == "" {
		if err := u.allocateNewAssetID(ctx, a, tags, promoteLogicalID); err != nil {
			return nil, mapCreateDBError(err)
		}
		return a, nil
	}
	if err := u.persistNewAsset(ctx, a, tags, promoteLogicalID); err != nil {
		if errors.Is(err, repository.ErrDuplicateAssetID) {
			return nil, ErrAssetIDTaken
		}
		return nil, mapCreateDBError(err)
	}
	return a, nil
}

// CreateChildAssetInput carries the fields needed for layered child-asset creation.
type CreateChildAssetInput struct {
	AssetType        string
	ParentAssetID    string
	StartTimestampNs int64
	EndTimestampNs   int64
	Metadata         map[string]interface{}
	SplitMethod      string
	SplitRunID       string
}

// splitMethodToRelation maps split_method to asset_relations.relation_type per
// the decision table in §3.2.1 of the hierarchy-and-derivatives design doc.
//
//	algo:*  → derived_from
//	manual / rule:* / ""  → split_from
func splitMethodToRelation(splitMethod string) string {
	if strings.HasPrefix(splitMethod, "algo:") {
		return "derived_from"
	}
	return "split_from"
}

// CreateChildAsset creates a child asset under a parent with automatic
// asset_relations edge insertion. This is the usecase behind the layered API
// (POST /assets/:id/{clips,actions,frames,tasks}).
func (u *Usecase) CreateChildAsset(ctx context.Context, in CreateChildAssetInput) (*models.Asset, error) {
	if in.EndTimestampNs <= in.StartTimestampNs {
		return nil, ErrInvalidRange
	}
	if in.ParentAssetID == "" {
		return nil, ErrNotFound
	}

	// Verify parent exists before proceeding.
	parent, err := u.repo.Get(ctx, in.ParentAssetID)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, ErrNotFound
	}

	// Build the asset model with a zero ID; InsertNew + prepAssetForWrite will
	// allocate timestamps and derive duration.
	// Inherit identity fields (McapFileID, SegmentLocator, RootAssetID) and
	// tenant/project scope from the parent — the DB CHECK constraints require these.
	a := &models.Asset{
		AssetType:        in.AssetType,
		ParentAssetID:    in.ParentAssetID,
		McapFileID:       parent.McapFileID,
		SegmentLocator:   parent.SegmentLocator,
		RootAssetID:      parent.RootAssetID,
		StartTimestampNs: in.StartTimestampNs,
		EndTimestampNs:   in.EndTimestampNs,
		DurationMs:       (in.EndTimestampNs - in.StartTimestampNs) / 1_000_000,
		SplitMethod:      in.SplitMethod,
		SplitRunID:       in.SplitRunID,
		Metadata:         in.Metadata,
		LifecycleMeta:    defaultLifecycleMeta(),
		RetentionTier:    parent.RetentionTier,
		TenantID:         parent.TenantID,
		ProjectID:        parent.ProjectID,
		Files:            map[string]string{},
		Tags:             map[string]string{},
		AlgoResults:      map[string]string{},
	}
	if u.schemaRegistry != nil {
		if err := u.schemaRegistry.Validate(a.AssetType, a.Metadata); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidTag, err.Error())
		}
	}

	// CYB-1164: validate hierarchy invariants before persisting.
	if u.validator != nil {
		if err := u.validator.ValidateCreate(ctx, a); err != nil {
			return nil, err
		}
	}

	relationType := splitMethodToRelation(in.SplitMethod)

	for range maxAssetIDAllocationAttempts {
		gid, err := id.GenerateAssetID()
		if err != nil {
			return nil, err
		}
		a.AssetID = gid
		a.Version = 0
		a.CreatedAt = time.Time{}

		err = u.withMutationTx(ctx, func(txCtx context.Context) error {
			if err := u.seedFirstVersion(txCtx, a); err != nil {
				return err
			}
			if err := u.repo.InsertNew(txCtx, a); err != nil {
				return err
			}
			if relRepo, ok := u.repo.(repository.AssetRelationWriter); ok {
				if err := relRepo.InsertRelation(txCtx, a.AssetID, in.ParentAssetID, relationType, in.SplitRunID); err != nil {
					return err
				}
			}
			if err := u.appendAssetEvent(txCtx, "asset_created", a, map[string]any{
				"asset_id":        a.AssetID,
				"asset_type":      a.AssetType,
				"parent_asset_id": in.ParentAssetID,
				"relation_type":   relationType,
			}); err != nil {
				return err
			}
			return u.seedInitialAlgoProjection(txCtx, a)
		})

		if err != nil {
			if errors.Is(err, repository.ErrDuplicateAssetID) {
				continue
			}
			return nil, mapCreateDBError(err)
		}
		return u.Get(ctx, a.AssetID)
	}
	return nil, fmt.Errorf("exhausted asset id allocation attempts (%d)", maxAssetIDAllocationAttempts)
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
	// LifecycleState is authoritative; Status is derived for API compat.
	// When both are provided, lifecycle_state wins.
	if in.LifecycleState != nil {
		ls := strings.TrimSpace(*in.LifecycleState)
		if ls == "" {
			return nil, fmt.Errorf("%w: lifecycle_state must not be empty", ErrInvalidTag)
		}
		if !models.IsValidLifecycleState(ls) {
			return nil, fmt.Errorf("%w: invalid lifecycle_state %q", ErrInvalidTag, ls)
		}
		a.LifecycleState = ls
		a.Status = models.AssetStatus(models.LifecycleToStatus(ls))
	} else if in.Status != nil {
		a.Status = models.AssetStatus(*in.Status)
		a.LifecycleState = models.StatusToLifecycle(*in.Status)
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
	// CYB-1070: customer.* namespace lint.
	if err := u.validateCustomerNamespace(ctx, in.Tags); err != nil {
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
		if (in.Status != nil || in.LifecycleState != nil) &&
			(prevStatus != string(a.Status) || prevLifecycle != a.LifecycleState) {
			if err := u.appendAssetEvent(txCtx, "asset_lifecycle_changed", a, map[string]any{
				"prev_status":          prevStatus,
				"new_status":           string(a.Status),
				"prev_lifecycle_state": prevLifecycle,
				"new_lifecycle_state":  a.LifecycleState,
			}); err != nil {
				return err
			}
		}
		return u.upsertTagProjection(txCtx, a, in.Tags, defaultTagSource())
	}); err != nil {
		return nil, err
	}
	return a, nil
}

func (u *Usecase) UpsertTag(ctx context.Context, assetID string, in UpsertTagInput) (*models.Asset, error) {
	if err := u.validateTags(map[string]string{in.Key: in.Value}); err != nil {
		return nil, err
	}
	// CYB-1070: customer.* namespace lint — verify the customer exists.
	if err := u.validateCustomerNamespace(ctx, map[string]string{in.Key: in.Value}); err != nil {
		return nil, err
	}
	src := tagSource{
		SourceType:    in.SourceType,
		SourceName:    in.SourceName,
		SourceVersion: in.SourceVersion,
		RunID:         in.RunID,
	}
	if src.SourceType == "" {
		src.SourceType = "human"
	}
	if err := u.validateTagSource(src.SourceType, src.SourceName, src.SourceVersion); err != nil {
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
		return u.upsertTagProjection(txCtx, a, map[string]string{in.Key: in.Value}, src)
	}); err != nil {
		return nil, err
	}
	return u.Get(ctx, assetID)
}

// findTagsForDelete returns the tag rows that will be removed by DeleteTag.
// When sourceType is empty all rows for the key are returned, matching the
// repo Delete behavior.
func (u *Usecase) findTagsForDelete(ctx context.Context, assetID, tagKey, sourceType string) ([]*models.AssetTag, error) {
	if u.tagRepo == nil {
		return nil, nil
	}
	rows, err := u.tagRepo.ListByAsset(ctx, assetID)
	if err != nil {
		return nil, err
	}
	var out []*models.AssetTag
	for _, row := range rows {
		if row.TagKey != tagKey {
			continue
		}
		if sourceType != "" && row.SourceType != sourceType {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

// DeleteTag removes tag rows for (assetID, tagKey). When sourceType is the
// empty string all sources for the key are removed; otherwise only the
// matching source is deleted.
func (u *Usecase) DeleteTag(ctx context.Context, assetID, tagKey, sourceType string) (*models.Asset, error) {
	a, err := u.repo.Get(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrNotFound
	}

	if err := u.withMutationTx(ctx, func(txCtx context.Context) error {
		victims, err := u.findTagsForDelete(txCtx, assetID, tagKey, sourceType)
		if err != nil {
			return err
		}
		if len(victims) == 0 {
			return nil
		}
		if err := u.tagRepo.Delete(txCtx, assetID, tagKey, sourceType); err != nil {
			return err
		}
		for _, v := range victims {
			if err := u.appendAssetEvent(txCtx, "tag_deleted", a, map[string]any{
				"tag_key":        v.TagKey,
				"tag_value":      v.TagValue,
				"tag_type":       v.TagType,
				"source_type":    v.SourceType,
				"source_name":    v.SourceName,
				"source_version": v.SourceVersion,
				"run_id":         v.RunID,
			}); err != nil {
				return err
			}
		}
		return nil
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
		if err := u.allocateNewAssetID(ctx, a, a.Tags, ""); err != nil {
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

// RecordAssetView increments view_count and updates last_viewed_at in
// asset_usage_stats (CYB-1095). Verifies the asset exists first.
func (u *Usecase) RecordAssetView(ctx context.Context, assetID string) error {
	a, err := u.repo.Get(ctx, assetID)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrNotFound
	}
	if u.usageStatsRepo == nil {
		return nil
	}
	return u.usageStatsRepo.RecordView(ctx, assetID)
}

// ToggleFavorite flips the favorite state for an asset and returns the new
// favorite_count (CYB-1096). Verifies the asset exists first.
func (u *Usecase) ToggleFavorite(ctx context.Context, assetID string) (int, error) {
	a, err := u.repo.Get(ctx, assetID)
	if err != nil {
		return 0, err
	}
	if a == nil {
		return 0, ErrNotFound
	}
	if u.usageStatsRepo == nil {
		return 0, fmt.Errorf("usage stats repository not configured")
	}
	return u.usageStatsRepo.ToggleFavorite(ctx, assetID)
}
