package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgconn"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/CyberOrigin2077/cyber-databrew/internal/models"
	"github.com/CyberOrigin2077/cyber-databrew/internal/repository"
)

// PipelineConfigRepo persists standalone config files.
type PipelineConfigRepo struct {
	c *Client
}

var _ repository.PipelineConfigRepository = (*PipelineConfigRepo)(nil)

// NewPipelineConfigRepo creates a repo bound to c.
func NewPipelineConfigRepo(c *Client) *PipelineConfigRepo {
	return &PipelineConfigRepo{c: c}
}

const pipelineConfigSelectCols = `id, name, description, owner, scope, tags, file_type, lifecycle, current_version, created_at, updated_at`

const pipelineConfigVersionSelectCols = `id, config_id, version, status, content, content_sha256, content_size_bytes, summary, author, created_at`

func scanPipelineConfig(rs rowScanner) (*models.PipelineConfig, error) {
	var (
		cfg     models.PipelineConfig
		tagsRaw []byte
	)
	if err := rs.Scan(
		&cfg.ID, &cfg.Name, &cfg.Description, &cfg.Owner, &cfg.Scope, &tagsRaw,
		&cfg.FileType, &cfg.Lifecycle, &cfg.CurrentVersion, &cfg.CreatedAt, &cfg.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(tagsRaw) > 0 {
		_ = json.Unmarshal(tagsRaw, &cfg.Tags)
	}
	if cfg.Tags == nil {
		cfg.Tags = []string{}
	}
	return &cfg, nil
}

func scanPipelineConfigVersion(rs rowScanner) (*models.PipelineConfigVersion, error) {
	var version models.PipelineConfigVersion
	if err := rs.Scan(
		&version.ID, &version.ConfigID, &version.Version, &version.Status, &version.Content,
		&version.ContentSHA256, &version.ContentSizeBytes, &version.Summary, &version.Author,
		&version.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &version, nil
}

func pipelineConfigVersionSummary(version models.PipelineConfigVersion) models.PipelineConfigVersion {
	version.Content = ""
	return version
}

func preparePipelineConfigForWrite(cfg *models.PipelineConfig, now time.Time) {
	if cfg.ID == "" {
		cfg.ID = uuid.NewString()
	}
	if cfg.Scope == "" {
		cfg.Scope = "dev"
	}
	if cfg.Lifecycle == "" {
		cfg.Lifecycle = "draft"
	}
	if cfg.FileType == "" {
		cfg.FileType = inferConfigFileType(cfg.Name)
	}
	if cfg.Tags == nil {
		cfg.Tags = []string{}
	}
	if cfg.CurrentVersion <= 0 {
		cfg.CurrentVersion = 1
	}
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now
}

func preparePipelineConfigVersionForWrite(version *models.PipelineConfigVersion, now time.Time) {
	if version.ID == "" {
		version.ID = uuid.NewString()
	}
	if version.Status == "" {
		version.Status = "draft"
	}
	if version.CreatedAt.IsZero() {
		version.CreatedAt = now
	}
	sum := sha256.Sum256([]byte(version.Content))
	version.ContentSHA256 = hex.EncodeToString(sum[:])
	version.ContentSizeBytes = len([]byte(version.Content))
}

func inferConfigFileType(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	if strings.HasSuffix(lower, ".json") {
		return "json"
	}
	return "yaml"
}

// Create stores config metadata and its first immutable file version.
func (r *PipelineConfigRepo) Create(ctx context.Context, cfg *models.PipelineConfig, version *models.PipelineConfigVersion) error {
	if cfg == nil || version == nil {
		return errors.New("postgres PipelineConfigRepo.Create: nil config or version")
	}
	return r.c.WithTx(ctx, func(txCtx context.Context) error {
		now := time.Now().UTC()
		preparePipelineConfigForWrite(cfg, now)
		version.ConfigID = cfg.ID
		version.Version = cfg.CurrentVersion
		preparePipelineConfigVersionForWrite(version, now)
		tags, _ := json.Marshal(cfg.Tags)
		db := dbFromCtx(txCtx, r.c.db)
		const configQ = `
INSERT INTO pipeline_configs (
  id, name, description, owner, scope, tags, file_type, lifecycle, current_version, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11
)`
		if err := db.Exec(txCtx, configQ,
			cfg.ID, cfg.Name, cfg.Description, cfg.Owner, cfg.Scope, tags, cfg.FileType,
			cfg.Lifecycle, cfg.CurrentVersion, cfg.CreatedAt, cfg.UpdatedAt,
		); err != nil {
			if isPgUniqueViolation(err) { return repository.ErrPipelineConfigNameExists }; return fmt.Errorf("postgres PipelineConfigRepo.Create config: %w", err)
		}
		const versionQ = `
INSERT INTO pipeline_config_versions (
  id, config_id, version, status, content, content_sha256, content_size_bytes, summary, author, created_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)`
		if err := db.Exec(txCtx, versionQ,
			version.ID, version.ConfigID, version.Version, version.Status, version.Content,
			version.ContentSHA256, version.ContentSizeBytes, version.Summary, version.Author,
			version.CreatedAt,
		); err != nil {
			return fmt.Errorf("postgres PipelineConfigRepo.Create version: %w", err)
		}
		return nil
	})
}

// FindAll lists config metadata and version counts without file contents.
func (r *PipelineConfigRepo) FindAll(ctx context.Context, filter *repository.PipelineConfigFilter) ([]models.PipelineConfig, error) {
	q := `SELECT ` + pipelineConfigSelectCols + `, COALESCE(version_count, 0)
FROM pipeline_configs pc
LEFT JOIN (
  SELECT config_id, COUNT(*) AS version_count
  FROM pipeline_config_versions
  GROUP BY config_id
) counts ON counts.config_id = pc.id`
	var args []any
	var conditions []string
	argIdx := 0
	if filter != nil {
		if filter.Query != "" {
			argIdx++
			conditions = append(conditions, fmt.Sprintf(`(
				pc.name ILIKE $%d OR pc.description ILIKE $%d OR pc.owner ILIKE $%d OR pc.file_type ILIKE $%d OR
				pc.tags::text ILIKE $%d
			)`, argIdx, argIdx, argIdx, argIdx, argIdx))
			args = append(args, "%"+filter.Query+"%")
		}
		if filter.Owner != "" {
			argIdx++
			conditions = append(conditions, fmt.Sprintf(`pc.owner = $%d`, argIdx))
			args = append(args, filter.Owner)
		}
		if filter.Scope != "" {
			argIdx++
			conditions = append(conditions, fmt.Sprintf(`pc.scope = $%d`, argIdx))
			args = append(args, filter.Scope)
		}
		if filter.Lifecycle != "" {
			argIdx++
			conditions = append(conditions, fmt.Sprintf(`pc.lifecycle = $%d`, argIdx))
			args = append(args, filter.Lifecycle)
		}
	}
	if len(conditions) > 0 {
		q += ` WHERE ` + strings.Join(conditions, ` AND `)
	}
	q += ` ORDER BY pc.updated_at DESC, pc.name ASC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres PipelineConfigRepo.FindAll: %w", err)
	}
	defer rows.Close()
	var out []models.PipelineConfig
	for rows.Next() {
		cfg, err := scanPipelineConfigWithVersionCount(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres PipelineConfigRepo.FindAll scan: %w", err)
		}
		out = append(out, *cfg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres PipelineConfigRepo.FindAll rows: %w", err)
	}
	return out, nil
}

func scanPipelineConfigWithVersionCount(rs rowScanner) (*models.PipelineConfig, error) {
	var (
		cfg     models.PipelineConfig
		tagsRaw []byte
	)
	if err := rs.Scan(
		&cfg.ID, &cfg.Name, &cfg.Description, &cfg.Owner, &cfg.Scope, &tagsRaw,
		&cfg.FileType, &cfg.Lifecycle, &cfg.CurrentVersion, &cfg.CreatedAt, &cfg.UpdatedAt,
		&cfg.VersionCount,
	); err != nil {
		return nil, err
	}
	if len(tagsRaw) > 0 {
		_ = json.Unmarshal(tagsRaw, &cfg.Tags)
	}
	if cfg.Tags == nil {
		cfg.Tags = []string{}
	}
	return &cfg, nil
}

// FindByID returns config metadata plus version summaries.
func (r *PipelineConfigRepo) FindByID(ctx context.Context, id string) (*models.PipelineConfig, error) {
	q := `SELECT ` + pipelineConfigSelectCols + `
FROM pipeline_configs
WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	cfg, err := scanPipelineConfig(db.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres PipelineConfigRepo.FindByID: %w", err)
	}
	versions, err := r.FindVersions(ctx, id)
	if err != nil {
		return nil, err
	}
	cfg.VersionCount = len(versions)
	cfg.Versions = versions
	return cfg, nil
}

// UpdateMetadata updates config metadata without mutating file versions.
func (r *PipelineConfigRepo) UpdateMetadata(ctx context.Context, cfg *models.PipelineConfig) error {
	if cfg == nil {
		return errors.New("postgres PipelineConfigRepo.UpdateMetadata: nil config")
	}
	cfg.UpdatedAt = time.Now().UTC()
	tags, _ := json.Marshal(cfg.Tags)
	const q = `
UPDATE pipeline_configs SET
  name = $2,
  description = $3,
  tags = $4::jsonb,
  file_type = $5,
  lifecycle = $6,
  updated_at = $7
WHERE id = $1`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.ExecResult(ctx, q, cfg.ID, cfg.Name, cfg.Description, tags, cfg.FileType, cfg.Lifecycle, cfg.UpdatedAt)
	if err != nil {
		if isPgUniqueViolation(err) { return repository.ErrPipelineConfigNameExists }; return fmt.Errorf("postgres PipelineConfigRepo.UpdateMetadata: %w", err)
	}
	if rows == 0 {
		return repository.ErrPipelineConfigNotFound
	}
	return nil
}

// UpdateVersionStatus updates the status of a specific version.
func (r *PipelineConfigRepo) UpdateVersionStatus(ctx context.Context, configID string, version int, status string) (*models.PipelineConfigVersion, error) {
	versionRow := &models.PipelineConfigVersion{}
	if err := r.c.db.QueryRow(ctx, `
UPDATE pipeline_config_versions
SET status = $3
WHERE config_id = $1 AND version = $2
RETURNING version, status, content, summary, author, created_at
`, configID, version, status).Scan(
		&versionRow.Version, &versionRow.Status, &versionRow.Content,
		&versionRow.Summary, &versionRow.Author, &versionRow.CreatedAt,
	); err != nil {
		if errors.Is(err, errNoRows) {
			return nil, repository.ErrPipelineConfigNotFound
		}
		return nil, fmt.Errorf("postgres PipelineConfigRepo.UpdateVersionStatus: %w", err)
	}
	return versionRow, nil
}

// CreateVersion appends a new immutable version and advances current_version.
func (r *PipelineConfigRepo) UpdateVersionContent(ctx context.Context, configID string, version int, content string, summary string) (*models.PipelineConfigVersion, error) {
	versionRow := &models.PipelineConfigVersion{}
	if err := r.c.db.QueryRow(ctx, `
UPDATE pipeline_config_versions
SET content = $3, summary = $4, content_sha256 = encode(sha256($3::bytea), 'hex'), content_size_bytes = length($3)
WHERE config_id = $1 AND version = $2
RETURNING version, status, content, summary, author, created_at
`, configID, version, content, summary).Scan(
		&versionRow.Version, &versionRow.Status, &versionRow.Content,
		&versionRow.Summary, &versionRow.Author, &versionRow.CreatedAt,
	); err != nil {
		if errors.Is(err, errNoRows) {
			return nil, repository.ErrPipelineConfigNotFound
		}
		return nil, fmt.Errorf("postgres PipelineConfigRepo.UpdateVersionContent: %w", err)
	}
	return versionRow, nil
}

func (r *PipelineConfigRepo) UpdateLifecycle(ctx context.Context, configID string, lifecycle string) error {
	err := r.c.db.Exec(ctx, `UPDATE pipeline_configs SET lifecycle = $2 WHERE id = $1`, configID, lifecycle)
	return err
}

func (r *PipelineConfigRepo) CreateVersion(ctx context.Context, configID string, version *models.PipelineConfigVersion) error {
	if version == nil {
		return errors.New("postgres PipelineConfigRepo.CreateVersion: nil version")
	}
	return r.c.WithTx(ctx, func(txCtx context.Context) error {
		db := dbFromCtx(txCtx, r.c.db)
		var nextVersion int
		if err := db.QueryRow(txCtx, `
SELECT COALESCE(MAX(version), 0) + 1
FROM pipeline_config_versions
	WHERE config_id = $1`, configID).Scan(&nextVersion); err != nil {
			return fmt.Errorf("postgres PipelineConfigRepo.CreateVersion next: %w", err)
		}
		if nextVersion == 1 {
			return repository.ErrPipelineConfigNotFound
		}
		now := time.Now().UTC()
		version.ConfigID = configID
		version.Version = nextVersion
		preparePipelineConfigVersionForWrite(version, now)
		const versionQ = `
INSERT INTO pipeline_config_versions (
  id, config_id, version, status, content, content_sha256, content_size_bytes, summary, author, created_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)`
		if err := db.Exec(txCtx, versionQ,
			version.ID, version.ConfigID, version.Version, version.Status, version.Content,
			version.ContentSHA256, version.ContentSizeBytes, version.Summary, version.Author,
			version.CreatedAt,
		); err != nil {
			return fmt.Errorf("postgres PipelineConfigRepo.CreateVersion insert: %w", err)
		}
		const configQ = `
UPDATE pipeline_configs
SET current_version = $2, lifecycle = $3, updated_at = $4
WHERE id = $1`
		rows, err := db.ExecResult(txCtx, configQ, configID, version.Version, version.Status, now)
		if err != nil {
			return fmt.Errorf("postgres PipelineConfigRepo.CreateVersion update config: %w", err)
		}
		if rows == 0 {
			return repository.ErrPipelineConfigNotFound
		}
		return nil
	})
}

// FindVersion returns one immutable file version including content.
func (r *PipelineConfigRepo) FindVersion(ctx context.Context, configID string, version int) (*models.PipelineConfigVersion, error) {
	q := `SELECT ` + pipelineConfigVersionSelectCols + `
FROM pipeline_config_versions
WHERE config_id = $1 AND version = $2`
	db := dbFromCtx(ctx, r.c.db)
	item, err := scanPipelineConfigVersion(db.QueryRow(ctx, q, configID, version))
	if err != nil {
		if errors.Is(err, errNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres PipelineConfigRepo.FindVersion: %w", err)
	}
	return item, nil
}

// FindVersions returns version summaries without file content.
func (r *PipelineConfigRepo) FindVersions(ctx context.Context, configID string) ([]models.PipelineConfigVersion, error) {
	q := `SELECT ` + pipelineConfigVersionSelectCols + `
FROM pipeline_config_versions
WHERE config_id = $1
ORDER BY version DESC`
	db := dbFromCtx(ctx, r.c.db)
	rows, err := db.Query(ctx, q, configID)
	if err != nil {
		return nil, fmt.Errorf("postgres PipelineConfigRepo.FindVersions: %w", err)
	}
	defer rows.Close()
	var out []models.PipelineConfigVersion
	for rows.Next() {
		version, err := scanPipelineConfigVersion(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres PipelineConfigRepo.FindVersions scan: %w", err)
		}
		out = append(out, pipelineConfigVersionSummary(*version))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres PipelineConfigRepo.FindVersions rows: %w", err)
	}
	return out, nil
}

// Deprecate marks a config and its current version as deprecated.
func (r *PipelineConfigRepo) Deprecate(ctx context.Context, id string) error {
	return r.c.WithTx(ctx, func(txCtx context.Context) error {
		db := dbFromCtx(txCtx, r.c.db)
		var currentVersion int
		if err := db.QueryRow(txCtx, `
UPDATE pipeline_configs
SET lifecycle = 'deprecated', updated_at = now()
WHERE id = $1
	RETURNING current_version`, id).Scan(&currentVersion); err != nil {
			if errors.Is(err, errNoRows) {
				return repository.ErrPipelineConfigNotFound
			}
			return fmt.Errorf("postgres PipelineConfigRepo.Deprecate config: %w", err)
		}
		if err := db.Exec(txCtx, `
UPDATE pipeline_config_versions
SET status = 'deprecated'
WHERE config_id = $1 AND version = $2`, id, currentVersion); err != nil {
			return fmt.Errorf("postgres PipelineConfigRepo.Deprecate version: %w", err)
		}
		return nil
	})
}

func isPgUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
