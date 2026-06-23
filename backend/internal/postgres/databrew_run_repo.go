package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

type DatabrewRunRepo struct {
	c *Client
}

func NewDatabrewRunRepo(c *Client) *DatabrewRunRepo { return &DatabrewRunRepo{c: c} }

var _ repository.DatabrewRunRepository = (*DatabrewRunRepo)(nil)

const databrewRunSelectCols = `id, type, name, status, runtime, runtime_namespace, runtime_resource_name,
runtime_uid, owner, created_by, message, created_at, started_at, finished_at, updated_at`

func scanDatabrewRun(rs rowScanner) (*models.DatabrewRun, error) {
	var run models.DatabrewRun
	if err := rs.Scan(
		&run.ID, &run.Type, &run.Name, &run.Status, &run.Runtime,
		&run.RuntimeNamespace, &run.RuntimeResourceName, &run.RuntimeUID,
		&run.Owner, &run.CreatedBy, &run.Message,
		&run.CreatedAt, &run.StartedAt, &run.FinishedAt, &run.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *DatabrewRunRepo) Save(ctx context.Context, run *models.DatabrewRun) error {
	if run == nil {
		return errors.New("postgres DatabrewRunRepo.Save: nil run")
	}
	now := time.Now().UTC()
	if run.ID == "" {
		run.ID = uuid.New().String()
	}
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now
	}
	run.UpdatedAt = now
	if run.Runtime == "" {
		run.Runtime = "argo"
	}
	const q = `
INSERT INTO databrew_runs (
  id, type, name, status, runtime, runtime_namespace, runtime_resource_name,
  runtime_uid, owner, created_by, message, created_at, started_at, finished_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
)
ON CONFLICT (id) DO UPDATE SET
  type = EXCLUDED.type,
  name = EXCLUDED.name,
  status = EXCLUDED.status,
  runtime = EXCLUDED.runtime,
  runtime_namespace = EXCLUDED.runtime_namespace,
  runtime_resource_name = EXCLUDED.runtime_resource_name,
  runtime_uid = EXCLUDED.runtime_uid,
  owner = EXCLUDED.owner,
  created_by = EXCLUDED.created_by,
  message = EXCLUDED.message,
  started_at = EXCLUDED.started_at,
  finished_at = EXCLUDED.finished_at,
  updated_at = EXCLUDED.updated_at`
	db := dbFromCtx(ctx, r.c.db)
	if err := db.Exec(ctx, q,
		run.ID, run.Type, run.Name, run.Status, run.Runtime,
		run.RuntimeNamespace, run.RuntimeResourceName, run.RuntimeUID,
		run.Owner, run.CreatedBy, run.Message,
		run.CreatedAt, run.StartedAt, run.FinishedAt, run.UpdatedAt,
	); err != nil {
		return fmt.Errorf("postgres DatabrewRunRepo.Save: %w", err)
	}
	return nil
}

func (r *DatabrewRunRepo) FindByID(ctx context.Context, id string) (*models.DatabrewRun, error) {
	const q = `SELECT ` + databrewRunSelectCols + ` FROM databrew_runs WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	row := db.QueryRow(ctx, q, id)
	run, err := scanDatabrewRun(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres DatabrewRunRepo.FindByID: %w", err)
	}
	return run, nil
}

func (r *DatabrewRunRepo) FindByRuntimeResource(ctx context.Context, namespace, resourceName string) (*models.DatabrewRun, error) {
	const q = `SELECT ` + databrewRunSelectCols + `
FROM databrew_runs
WHERE runtime_namespace = $1 AND runtime_resource_name = $2
LIMIT 1`
	db := dbFromCtx(ctx, r.c.db)
	row := db.QueryRow(ctx, q, namespace, resourceName)
	run, err := scanDatabrewRun(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres DatabrewRunRepo.FindByRuntimeResource: %w", err)
	}
	return run, nil
}

func (r *DatabrewRunRepo) List(ctx context.Context, filter models.DatabrewRunListFilter) ([]models.DatabrewRun, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	offset := (page - 1) * pageSize

	var (
		conds  []string
		args   []any
		argPos = 1
	)
	if filter.Type != "" {
		conds = append(conds, fmt.Sprintf("type = $%d", argPos))
		args = append(args, filter.Type)
		argPos++
	}
	if filter.Status != "" {
		conds = append(conds, fmt.Sprintf("status = $%d", argPos))
		args = append(args, filter.Status)
		argPos++
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	db := dbFromCtx(ctx, r.c.db)
	countQ := `SELECT COUNT(*) FROM databrew_runs ` + where
	var total int
	if err := db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres DatabrewRunRepo.List count: %w", err)
	}

	listQ := fmt.Sprintf(`SELECT %s FROM databrew_runs %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		databrewRunSelectCols, where, argPos, argPos+1)
	args = append(args, pageSize, offset)
	rows, err := db.Query(ctx, listQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres DatabrewRunRepo.List query: %w", err)
	}
	defer rows.Close()

	items := make([]models.DatabrewRun, 0, pageSize)
	for rows.Next() {
		run, err := scanDatabrewRun(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres DatabrewRunRepo.List scan: %w", err)
		}
		items = append(items, *run)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("postgres DatabrewRunRepo.List rows: %w", err)
	}
	return items, total, nil
}

type ComponentBuildRunRepo struct{ c *Client }

func NewComponentBuildRunRepo(c *Client) *ComponentBuildRunRepo {
	return &ComponentBuildRunRepo{c: c}
}

var _ repository.ComponentBuildRunRepository = (*ComponentBuildRunRepo)(nil)

func (r *ComponentBuildRunRepo) Save(ctx context.Context, row *models.ComponentBuildRun) error {
	if row == nil || row.RunID == "" {
		return errors.New("postgres ComponentBuildRunRepo.Save: missing run_id")
	}
	const q = `
INSERT INTO component_build_runs (
  run_id, component_id, repo_url, git_ref, commit_sha, dockerfile,
  build_context, image_repository, image_tag, image_digest
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (run_id) DO UPDATE SET
  component_id = EXCLUDED.component_id,
  repo_url = EXCLUDED.repo_url,
  git_ref = EXCLUDED.git_ref,
  commit_sha = EXCLUDED.commit_sha,
  dockerfile = EXCLUDED.dockerfile,
  build_context = EXCLUDED.build_context,
  image_repository = EXCLUDED.image_repository,
  image_tag = EXCLUDED.image_tag,
  image_digest = EXCLUDED.image_digest`
	db := dbFromCtx(ctx, r.c.db)
	return db.Exec(ctx, q,
		row.RunID, row.ComponentID, row.RepoURL, row.GitRef, row.CommitSHA, row.Dockerfile,
		row.BuildContext, row.ImageRepository, row.ImageTag, row.ImageDigest,
	)
}

func (r *ComponentBuildRunRepo) FindByRunID(ctx context.Context, runID string) (*models.ComponentBuildRun, error) {
	const q = `SELECT run_id, component_id, repo_url, git_ref, commit_sha, dockerfile,
build_context, image_repository, image_tag, image_digest
FROM component_build_runs WHERE run_id = $1`
	db := dbFromCtx(ctx, r.c.db)
	var row models.ComponentBuildRun
	err := db.QueryRow(ctx, q, runID).Scan(
		&row.RunID, &row.ComponentID, &row.RepoURL, &row.GitRef, &row.CommitSHA, &row.Dockerfile,
		&row.BuildContext, &row.ImageRepository, &row.ImageTag, &row.ImageDigest,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

type RAGBuildRunRepo struct{ c *Client }

func NewRAGBuildRunRepo(c *Client) *RAGBuildRunRepo { return &RAGBuildRunRepo{c: c} }

var _ repository.RAGBuildRunRepository = (*RAGBuildRunRepo)(nil)

func (r *RAGBuildRunRepo) Save(ctx context.Context, row *models.RAGBuildRun) error {
	if row == nil || row.RunID == "" {
		return errors.New("postgres RAGBuildRunRepo.Save: missing run_id")
	}
	snapshot, err := marshalMapForJSONB(row.DatasourceSnapshot)
	if err != nil {
		return err
	}
	const q = `
INSERT INTO rag_build_runs (
  run_id, knowledge_base_id, datasource_snapshot, embedding_model, vector_index_name, release_version
) VALUES ($1,$2,$3::jsonb,$4,$5,$6)
ON CONFLICT (run_id) DO UPDATE SET
  knowledge_base_id = EXCLUDED.knowledge_base_id,
  datasource_snapshot = EXCLUDED.datasource_snapshot,
  embedding_model = EXCLUDED.embedding_model,
  vector_index_name = EXCLUDED.vector_index_name,
  release_version = EXCLUDED.release_version`
	db := dbFromCtx(ctx, r.c.db)
	return db.Exec(ctx, q,
		row.RunID, row.KnowledgeBaseID, snapshot, row.EmbeddingModel, row.VectorIndexName, row.ReleaseVersion,
	)
}

func (r *RAGBuildRunRepo) FindByRunID(ctx context.Context, runID string) (*models.RAGBuildRun, error) {
	const q = `SELECT run_id, knowledge_base_id, datasource_snapshot, embedding_model, vector_index_name, release_version
FROM rag_build_runs WHERE run_id = $1`
	db := dbFromCtx(ctx, r.c.db)
	var (
		row        models.RAGBuildRun
		snapshot   []byte
	)
	err := db.QueryRow(ctx, q, runID).Scan(
		&row.RunID, &row.KnowledgeBaseID, &snapshot, &row.EmbeddingModel, &row.VectorIndexName, &row.ReleaseVersion,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if len(snapshot) > 0 {
		_ = json.Unmarshal(snapshot, &row.DatasourceSnapshot)
	}
	if row.DatasourceSnapshot == nil {
		row.DatasourceSnapshot = map[string]interface{}{}
	}
	return &row, nil
}

type ComponentReleaseRepo struct{ c *Client }

func NewComponentReleaseRepo(c *Client) *ComponentReleaseRepo { return &ComponentReleaseRepo{c: c} }

var _ repository.ComponentReleaseRepository = (*ComponentReleaseRepo)(nil)

func (r *ComponentReleaseRepo) Save(ctx context.Context, release *models.ComponentRelease) error {
	if release == nil {
		return errors.New("postgres ComponentReleaseRepo.Save: nil release")
	}
	now := time.Now().UTC()
	if release.ID == "" {
		release.ID = uuid.New().String()
	}
	if release.CreatedAt.IsZero() {
		release.CreatedAt = now
	}
	const q = `
INSERT INTO component_releases (
  id, component_id, source_commit, image, image_tag, image_digest, release_label, build_run_id, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (id) DO UPDATE SET
  source_commit = EXCLUDED.source_commit,
  image = EXCLUDED.image,
  image_tag = EXCLUDED.image_tag,
  image_digest = EXCLUDED.image_digest,
  release_label = EXCLUDED.release_label,
  build_run_id = EXCLUDED.build_run_id`
	db := dbFromCtx(ctx, r.c.db)
	return db.Exec(ctx, q,
		release.ID, release.ComponentID, release.SourceCommit, release.Image,
		release.ImageTag, release.ImageDigest, release.ReleaseLabel, release.BuildRunID, release.CreatedAt,
	)
}

func (r *ComponentReleaseRepo) ListByComponentID(ctx context.Context, componentID string) ([]models.ComponentRelease, error) {
	const q = `SELECT id, component_id, source_commit, image, image_tag, image_digest, release_label, build_run_id, created_at
FROM component_releases
WHERE component_id = $1
ORDER BY created_at DESC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, componentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.ComponentRelease, 0)
	for rows.Next() {
		var item models.ComponentRelease
		if err := rows.Scan(
			&item.ID, &item.ComponentID, &item.SourceCommit, &item.Image,
			&item.ImageTag, &item.ImageDigest, &item.ReleaseLabel, &item.BuildRunID, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
