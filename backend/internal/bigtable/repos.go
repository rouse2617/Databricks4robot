package bigtable

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigtable"

	"data-platform/internal/models"
	"data-platform/internal/repository"
)

// AssetRepo provides read/write access to the `assets` Bigtable table.
type AssetRepo struct {
	table    btTable
	idxTable btTable
}

// Ensure Bigtable implementation satisfies repository abstraction.
var _ repository.AssetRepository = (*AssetRepo)(nil)

func NewAssetRepo(c *Client) *AssetRepo {
	return &AssetRepo{
		table:    c.Table(TableAssets),
		idxTable: c.Table(TableIdxSegmentsByFile),
	}
}

// Get fetches a single asset by ID.
func (r *AssetRepo) Get(ctx context.Context, assetID string) (*models.Asset, error) {
	row, err := r.table.ReadRow(ctx, AssetKey(assetID),
		bigtable.RowFilter(bigtable.LatestNFilter(1)),
	)
	if err != nil {
		return nil, fmt.Errorf("AssetRepo.Get: %w", err)
	}
	if row == nil {
		return nil, nil
	}
	return rowToAsset(row), nil
}

// Set writes all columns of an asset (full overwrite / upsert).
func (r *AssetRepo) Set(ctx context.Context, a *models.Asset) error {
	now := time.Now()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	a.UpdatedAt = now
	a.Version++

	mut := bigtable.NewMutation()

	ts := bigtable.Now()
	cf := CFMeta

	mut.Set(cf, "mcap_file_id", ts, B(a.McapFileID))
	mut.Set(cf, "start_timestamp_ns", ts, PackInt64(a.StartTimestampNs))
	mut.Set(cf, "end_timestamp_ns", ts, PackInt64(a.EndTimestampNs))
	mut.Set(cf, "duration_sec", ts, B(strconv.FormatFloat(a.DurationSec, 'f', 6, 64)))
	mut.Set(cf, "reviewer", ts, B(a.Reviewer))
	mut.Set(cf, "status", ts, B(string(a.Status)))
	mut.Set(cf, "owner", ts, B(a.Owner))
	mut.Set(cf, "type", ts, B(a.SegType))
	mut.Set(cf, "env", ts, B(a.Env))
	mut.Set(cf, "task", ts, B(a.Task))
	mut.Set(cf, "delivery_count", ts, PackInt64(int64(a.DeliveryCount)))
	mut.Set(cf, "last_delivered_to", ts, B(a.LastDeliveredTo))
	if a.LastDeliveredAt != nil {
		mut.Set(cf, "last_delivered_at", ts, RFC3339(*a.LastDeliveredAt))
	}
	mut.Set(cf, "created_at", ts, RFC3339(a.CreatedAt))
	mut.Set(cf, "updated_at", ts, RFC3339(a.UpdatedAt))
	mut.Set(cf, "version", ts, PackInt64(a.Version))

	for k, v := range a.AlgoResults {
		mut.Set(CFAlgo, k, ts, B(v))
	}
	for k, v := range a.Tags {
		mut.Set(CFTag, k, ts, B(v))
	}
	for k, v := range a.Files {
		mut.Set(CFFiles, k, ts, B(v))
	}
	for k, v := range a.LifecycleMeta {
		mut.Set(cf, k, ts, B(fmt.Sprintf("%v", v)))
	}

	if err := r.table.Apply(ctx, AssetKey(a.AssetID), mut); err != nil {
		return fmt.Errorf("AssetRepo.Set: %w", err)
	}
	return nil
}

// WriteSegmentIndex writes one row to idx_segments_by_file.
// Row key pattern: <mcap_file_id>#<start_timestamp_ns>#<asset_id>.
func (r *AssetRepo) WriteSegmentIndex(ctx context.Context, a *models.Asset) error {
	mut := bigtable.NewMutation()
	mut.Set(CFRef, "asset_id", bigtable.Now(), B(a.AssetID))
	if err := r.idxTable.Apply(ctx, IdxSegmentKey(a.McapFileID, a.StartTimestampNs, a.AssetID), mut); err != nil {
		return fmt.Errorf("AssetRepo.WriteSegmentIndex: %w", err)
	}
	return nil
}

// SoftDelete marks an asset as archived (status=archived) without removing it.
func (r *AssetRepo) SoftDelete(ctx context.Context, assetID string) error {
	mut := bigtable.NewMutation()
	mut.Set(CFMeta, "status", bigtable.Now(), B(string(models.AssetStatusArchived)))
	mut.Set(CFMeta, "updated_at", bigtable.Now(), RFC3339(time.Now()))
	if err := r.table.Apply(ctx, AssetKey(assetID), mut); err != nil {
		return fmt.Errorf("AssetRepo.SoftDelete: %w", err)
	}
	return nil
}

// ListWithFilters scans assets with early termination to avoid full-table scans.
//
// Optimization: when there are no filter conditions, we stop scanning as soon as
// we have enough rows to fill the requested page. With filters, we scan up to a
// configurable limit (maxScanRows) and apply filters in memory.
func (r *AssetRepo) ListWithFilters(ctx context.Context, whereSQL string, args []interface{},
	page, pageSize int, orderBy string) ([]*models.Asset, int64, error) {

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	conditions := parseWhereSQL(whereSQL, args)
	hasFilters := len(conditions) > 0

	// How many matching rows we need: enough to fill the requested page.
	needed := page * pageSize
	// Max rows to scan from Bigtable (safety cap to prevent runaway scans).
	// With filters we need to scan more because many rows may not match.
	maxScan := needed * 5
	if hasFilters {
		maxScan = needed * 20
	}
	if maxScan < 500 {
		maxScan = 500
	}
	if maxScan > 50000 {
		maxScan = 50000
	}

	scanCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var matched []*models.Asset
	scanned := 0
	earlyStop := false

	scanErr := r.table.ReadRows(scanCtx, bigtable.PrefixRange(RowKeyPrefix), func(row bigtable.Row) bool {
		scanned++
		a := rowToAsset(row)

		// Skip archived.
		if a.Status == models.AssetStatusArchived {
			return scanned < maxScan
		}

		// Apply filters inline (avoid collecting all rows first).
		if hasFilters {
			pass := true
			for _, cond := range conditions {
				if !matchCondition(a, cond) {
					pass = false
					break
				}
			}
			if !pass {
				return scanned < maxScan
			}
		}

		matched = append(matched, a)

		// Early termination: if no sorting needed and we have enough for the page,
		// we can stop. With sorting we need all matching rows up to maxScan.
		if orderBy == "" && !hasFilters && len(matched) >= needed {
			earlyStop = true
			return false
		}

		return scanned < maxScan
	}, bigtable.RowFilter(bigtable.LatestNFilter(1)))

	if scanErr != nil && scanCtx.Err() == nil {
		return nil, 0, fmt.Errorf("AssetRepo.ListWithFilters: %w", scanErr)
	}

	total := int64(len(matched))
	// If we hit the scan limit or early-stopped, total is approximate.
	if scanned >= maxScan || earlyStop {
		// We don't know the true total — report what we found as a lower bound.
		// The "+" suffix convention isn't possible in int64, so we just report as-is.
	}

	// Sort in-memory.
	if orderBy != "" {
		sortAssets(matched, orderBy)
	}

	// Paginate.
	start := (page - 1) * pageSize
	if start >= len(matched) {
		return []*models.Asset{}, total, nil
	}
	end := start + pageSize
	if end > len(matched) {
		end = len(matched)
	}
	return matched[start:end], total, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// whereSQL parsing and in-memory filtering helpers
// ──────────────────────────────────────────────────────────────────────────────

// parsedCondition represents a single filter condition extracted from whereSQL.
type parsedCondition struct {
	field    string      // e.g. "status", "cf_algo.sam2@1.0:status"
	op       string      // SQL operator: "=", "!=", "<", ">", "<=", ">=", "LIKE", "ILIKE", "IN", "NOT IN"
	value    interface{} // the resolved value from args
	isJsonb  bool        // whether this is a JSONB path field
	jsonbCol string      // JSONB column name (e.g. "cf_algo")
	jsonbKey string      // JSONB key (e.g. "sam2@1.0:status")
}

// conditionPattern matches patterns like: field OP $N
// where field can be a quoted identifier, a JSONB extraction expression, or a cast expression.
var conditionPattern = regexp.MustCompile(
	`(?:` +
		`\(([^)]+)\)::(?:NUMERIC|TIMESTAMPTZ)` + // group 1: cast expression like (cf_algo#>>'{key}')::NUMERIC
		`|` +
		`([^\s]+)` + // group 2: simple field or JSONB expression
		`)` +
		`\s+` +
		`(=|!=|<=|>=|<>|<|>|LIKE|ILIKE|NOT\s+IN|IN|@>|IS\s+NULL|IS\s+NOT\s+NULL)` + // group 3: operator
		`\s*` +
		`(?:\(([^)]*)\)|\$(\d+)(?:::jsonb)?)?`, // group 4: IN list or group 5: param number
)

// parseWhereSQL parses a whereSQL string into structured conditions.
func parseWhereSQL(whereSQL string, args []interface{}) []parsedCondition {
	if whereSQL == "" {
		return nil
	}

	// Split on " AND " but respect parentheses depth to avoid splitting
	// inside subqueries like EXISTS (SELECT ... WHERE ... AND ...).
	parts := splitWhereAND(whereSQL)
	var conditions []parsedCondition

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "TRUE" || part == "FALSE" {
			continue
		}

		// Handle virtual field: algo_status EXISTS pattern.
		// Pattern: EXISTS (SELECT 1 FROM jsonb_each_text(cf_algo) WHERE key LIKE '%:status' AND value = $N)
		if strings.HasPrefix(part, "EXISTS (SELECT 1 FROM jsonb_each_text(cf_algo)") {
			paramRe := regexp.MustCompile(`\$(\d+)`)
			m := paramRe.FindStringSubmatch(part)
			if m != nil {
				idx, _ := strconv.Atoi(m[1])
				var value interface{}
				if idx >= 1 && idx <= len(args) {
					value = args[idx-1]
				}
				conditions = append(conditions, parsedCondition{
					field: "algo_status",
					op:    "=",
					value: value,
				})
			}
			continue
		}

		// Handle virtual field: has:delivery patterns.
		if part == "delivery_count > 0" {
			conditions = append(conditions, parsedCondition{
				field: "has:delivery",
				op:    "=",
				value: true,
			})
			continue
		}
		if part == "delivery_count = 0" {
			conditions = append(conditions, parsedCondition{
				field: "has:delivery",
				op:    "=",
				value: false,
			})
			continue
		}

		matches := conditionPattern.FindStringSubmatch(part)
		if matches == nil {
			continue // skip unparseable conditions (lenient mode)
		}

		// Extract field expression.
		fieldExpr := matches[1] // from cast expression
		if fieldExpr == "" {
			fieldExpr = matches[2] // from simple field
		}
		op := strings.TrimSpace(matches[3])
		inList := matches[4]
		paramStr := matches[5]

		// Normalize operator.
		op = strings.ToUpper(strings.Join(strings.Fields(op), " "))

		// Handle IS NULL / IS NOT NULL.
		if op == "IS NULL" || op == "IS NOT NULL" {
			cond := parsedCondition{op: op}
			resolveFieldExpr(fieldExpr, &cond)
			conditions = append(conditions, cond)
			continue
		}

		// Resolve value from args.
		var value interface{}
		if op == "IN" || op == "NOT IN" {
			// Parse parameter references from the IN list.
			value = resolveInValues(inList, args)
		} else if paramStr != "" {
			idx, _ := strconv.Atoi(paramStr)
			if idx >= 1 && idx <= len(args) {
				value = args[idx-1]
			}
		}

		cond := parsedCondition{op: op, value: value}
		resolveFieldExpr(fieldExpr, &cond)
		conditions = append(conditions, cond)
	}

	return conditions
}

// splitWhereAND splits a WHERE clause on " AND " while respecting parentheses depth.
// This prevents splitting inside subqueries like EXISTS (SELECT ... WHERE ... AND ...).
func splitWhereAND(sql string) []string {
	var parts []string
	depth := 0
	start := 0
	sep := " AND "
	for i := 0; i < len(sql); i++ {
		switch sql[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		}
		if depth == 0 && i+len(sep) <= len(sql) && sql[i:i+len(sep)] == sep {
			parts = append(parts, sql[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	parts = append(parts, sql[start:])
	return parts
}

// resolveInValues extracts values from an IN clause parameter list like "$1, $2, $3".
func resolveInValues(inList string, args []interface{}) []interface{} {
	paramRe := regexp.MustCompile(`\$(\d+)`)
	matches := paramRe.FindAllStringSubmatch(inList, -1)
	var values []interface{}
	for _, m := range matches {
		idx, _ := strconv.Atoi(m[1])
		if idx >= 1 && idx <= len(args) {
			values = append(values, args[idx-1])
		}
	}
	return values
}

// resolveFieldExpr parses a field expression and populates the condition's field info.
// Handles: "status", cf_algo#>>'{sam2@1.0,status}', "cf_algo"."key"
func resolveFieldExpr(expr string, cond *parsedCondition) {
	// Strip surrounding quotes from simple fields.
	expr = strings.Trim(expr, `"`)

	// Check for JSONB extraction: cf_algo#>>'{key1,key2}'
	if idx := strings.Index(expr, "#>>"); idx >= 0 {
		col := strings.Trim(expr[:idx], `"`)
		pathPart := expr[idx+3:]
		pathPart = strings.Trim(pathPart, "'{}")
		keys := strings.Split(pathPart, ",")
		cond.isJsonb = true
		cond.jsonbCol = col
		cond.jsonbKey = strings.Join(keys, ".")
		cond.field = col + "." + cond.jsonbKey
		return
	}

	// Check for JSONB arrow operator: cf_algo->>'key'
	if idx := strings.Index(expr, "->>"); idx >= 0 {
		col := strings.Trim(expr[:idx], `"`)
		key := strings.Trim(expr[idx+3:], "' \"")
		cond.isJsonb = true
		cond.jsonbCol = col
		cond.jsonbKey = key
		cond.field = col + "." + key
		return
	}

	cond.field = expr
}

// getAssetFieldValue returns the string value of an asset field for comparison.
func getAssetFieldValue(a *models.Asset, cond *parsedCondition) (string, bool) {
	if cond.isJsonb {
		return getJsonbFieldValue(a, cond.jsonbCol, cond.jsonbKey)
	}

	switch cond.field {
	case "status":
		return string(a.Status), true
	case "owner":
		return a.Owner, true
	case "reviewer":
		return a.Reviewer, true
	case "env":
		return a.Env, true
	case "task":
		return a.Task, true
	case "type":
		return a.SegType, true
	case "mcap_file_id":
		return a.McapFileID, true
	case "asset_id":
		return a.AssetID, true
	case "created_at":
		return a.CreatedAt.Format(time.RFC3339Nano), true
	case "updated_at":
		return a.UpdatedAt.Format(time.RFC3339Nano), true
	case "start_timestamp_ns":
		return strconv.FormatInt(a.StartTimestampNs, 10), true
	case "end_timestamp_ns":
		return strconv.FormatInt(a.EndTimestampNs, 10), true
	case "duration_sec":
		return strconv.FormatFloat(a.DurationSec, 'f', -1, 64), true
	case "delivery_count":
		return strconv.Itoa(a.DeliveryCount), true
	case "version":
		return strconv.FormatInt(a.Version, 10), true
	default:
		return "", false
	}
}

// getJsonbFieldValue returns the value from a JSONB-mapped field.
func getJsonbFieldValue(a *models.Asset, col, key string) (string, bool) {
	switch col {
	case "cf_meta":
		return getMetaFieldValue(a, key)
	case "cf_algo":
		v, ok := a.AlgoResults[key]
		return v, ok
	case "cf_tag":
		v, ok := a.Tags[key]
		return v, ok
	case "cf_files":
		v, ok := a.Files[key]
		return v, ok
	default:
		return "", false
	}
}

func getMetaFieldValue(a *models.Asset, key string) (string, bool) {
	switch key {
	case "end_timestamp_ns":
		return strconv.FormatInt(a.EndTimestampNs, 10), true
	case "duration_sec":
		return strconv.FormatFloat(a.DurationSec, 'f', -1, 64), true
	case "reviewer":
		return a.Reviewer, true
	case "owner":
		return a.Owner, true
	case "type":
		return a.SegType, true
	case "env":
		return a.Env, true
	case "task":
		return a.Task, true
	case "delivery_count":
		return strconv.Itoa(a.DeliveryCount), true
	case "last_delivered_to":
		return a.LastDeliveredTo, true
	case "last_delivered_at":
		if a.LastDeliveredAt == nil {
			return "", false
		}
		return a.LastDeliveredAt.Format(time.RFC3339Nano), true
	default:
		if a.LifecycleMeta == nil {
			return "", false
		}
		v, ok := a.LifecycleMeta[key]
		if !ok || v == nil {
			return "", false
		}
		switch typed := v.(type) {
		case string:
			return typed, true
		case time.Time:
			return typed.Format(time.RFC3339Nano), true
		default:
			return fmt.Sprintf("%v", typed), true
		}
	}
}

// applyCondition filters assets based on a single parsed condition.
func applyCondition(assets []*models.Asset, cond parsedCondition) []*models.Asset {
	var result []*models.Asset
	for _, a := range assets {
		if matchCondition(a, cond) {
			result = append(result, a)
		}
	}
	return result
}

// matchCondition checks if a single asset matches a condition.
func matchCondition(a *models.Asset, cond parsedCondition) bool {
	// Handle virtual fields: algo_status and has:delivery.
	if cond.field == "algo_status" {
		// Scan all cf_algo keys ending in ":status" and match if any value equals the filter value.
		target := fmt.Sprintf("%v", cond.value)
		for k, v := range a.AlgoResults {
			if strings.HasSuffix(k, ":status") && v == target {
				return true
			}
		}
		return false
	}
	if cond.field == "has:delivery" || cond.field == "has_delivery" {
		// Translate to delivery_count > 0.
		boolVal := true
		switch v := cond.value.(type) {
		case bool:
			boolVal = v
		case string:
			boolVal = v != "false" && v != "0"
		}
		if boolVal {
			return a.DeliveryCount > 0
		}
		return a.DeliveryCount == 0
	}

	fieldVal, exists := getAssetFieldValue(a, &cond)

	switch cond.op {
	case "IS NULL":
		return !exists || fieldVal == ""
	case "IS NOT NULL":
		return exists && fieldVal != ""
	}

	if !exists {
		return false
	}

	valueStr := fmt.Sprintf("%v", cond.value)

	switch cond.op {
	case "=":
		return fieldVal == valueStr
	case "!=", "<>":
		return fieldVal != valueStr
	case "<":
		return compareValues(fieldVal, valueStr) < 0
	case ">":
		return compareValues(fieldVal, valueStr) > 0
	case "<=":
		return compareValues(fieldVal, valueStr) <= 0
	case ">=":
		return compareValues(fieldVal, valueStr) >= 0
	case "LIKE":
		return matchLike(fieldVal, valueStr, true)
	case "ILIKE":
		return matchLike(fieldVal, valueStr, false)
	case "IN":
		return matchIn(fieldVal, cond.value, false)
	case "NOT IN":
		return matchIn(fieldVal, cond.value, true)
	default:
		return true // unknown operator → lenient mode, don't filter
	}
}

// compareValues compares two string values, attempting numeric comparison first.
func compareValues(a, b string) int {
	// Try numeric comparison.
	af, aErr := strconv.ParseFloat(a, 64)
	bf, bErr := strconv.ParseFloat(b, 64)
	if aErr == nil && bErr == nil {
		if af < bf {
			return -1
		}
		if af > bf {
			return 1
		}
		return 0
	}

	// Try time comparison.
	at, aErr := time.Parse(time.RFC3339Nano, a)
	bt, bErr := time.Parse(time.RFC3339Nano, b)
	if aErr == nil && bErr == nil {
		if at.Before(bt) {
			return -1
		}
		if at.After(bt) {
			return 1
		}
		return 0
	}

	// Fall back to string comparison.
	return strings.Compare(a, b)
}

// matchLike implements SQL LIKE pattern matching with % and _ wildcards.
func matchLike(value, pattern string, caseSensitive bool) bool {
	if !caseSensitive {
		value = strings.ToLower(value)
		pattern = strings.ToLower(pattern)
	}

	// Convert SQL LIKE pattern to a simple matching approach.
	// % matches any sequence, _ matches any single character.
	return matchLikeRecursive(value, pattern)
}

func matchLikeRecursive(s, p string) bool {
	for len(p) > 0 {
		switch p[0] {
		case '%':
			// Skip consecutive %
			for len(p) > 0 && p[0] == '%' {
				p = p[1:]
			}
			if len(p) == 0 {
				return true
			}
			for i := 0; i <= len(s); i++ {
				if matchLikeRecursive(s[i:], p) {
					return true
				}
			}
			return false
		case '_':
			if len(s) == 0 {
				return false
			}
			s = s[1:]
			p = p[1:]
		default:
			if len(s) == 0 || s[0] != p[0] {
				return false
			}
			s = s[1:]
			p = p[1:]
		}
	}
	return len(s) == 0
}

// matchIn checks if a value is in (or not in) a list.
func matchIn(fieldVal string, value interface{}, negate bool) bool {
	arr, ok := value.([]interface{})
	if !ok {
		// Single value comparison.
		match := fieldVal == fmt.Sprintf("%v", value)
		if negate {
			return !match
		}
		return match
	}

	for _, v := range arr {
		if fieldVal == fmt.Sprintf("%v", v) {
			return !negate // found: IN→true, NOT IN→false
		}
	}
	return negate // not found: IN→false, NOT IN→true
}

// sortAssets sorts assets in-memory based on an orderBy expression.
// Format: "field ASC" or "field DESC" or JSONB path expression.
func sortAssets(assets []*models.Asset, orderBy string) {
	if len(assets) <= 1 {
		return
	}

	parts := strings.Fields(orderBy)
	if len(parts) == 0 {
		return
	}

	fieldExpr := parts[0]
	direction := "ASC"
	if len(parts) > 1 {
		direction = strings.ToUpper(parts[len(parts)-1])
	}
	desc := direction == "DESC"

	// Build a pseudo-condition to reuse field resolution logic.
	cond := &parsedCondition{}
	resolveFieldExpr(fieldExpr, cond)

	sort.SliceStable(assets, func(i, j int) bool {
		vi, _ := getAssetFieldValue(assets[i], cond)
		vj, _ := getAssetFieldValue(assets[j], cond)
		cmp := compareValues(vi, vj)
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})
}

// MergeCfAlgo atomically merges cf:algo and cf:files fields using optimistic locking.
// It verifies the row exists, builds a mutation for algo/files KV pairs (setting non-nil
// values and deleting nil values), increments the version, and uses CheckAndMutateRow
// with a ValueRangeFilter for exact version matching.
func (r *AssetRepo) MergeCfAlgo(ctx context.Context, assetID string, expectedVersion int64,
	algoKV map[string]interface{}, filesKV map[string]interface{}) (int64, error) {

	// 1. Verify row exists.
	row, err := r.table.ReadRow(ctx, AssetKey(assetID), bigtable.RowFilter(bigtable.LatestNFilter(1)))
	if err != nil {
		return 0, fmt.Errorf("AssetRepo.MergeCfAlgo: read: %w", err)
	}
	if row == nil {
		return 0, fmt.Errorf("AssetRepo.MergeCfAlgo: asset %s not found", assetID)
	}

	// 2. Build mutation.
	mut := bigtable.NewMutation()
	ts := bigtable.Now()

	for k, v := range algoKV {
		if v != nil {
			mut.Set(CFAlgo, k, ts, B(fmt.Sprintf("%v", v)))
		} else {
			mut.DeleteCellsInColumn(CFAlgo, k)
		}
	}

	for k, v := range filesKV {
		if v != nil {
			mut.Set(CFFiles, k, ts, B(fmt.Sprintf("%v", v)))
		} else {
			mut.DeleteCellsInColumn(CFFiles, k)
		}
	}

	// Increment version and update timestamp in the same mutation.
	mut.Set(CFMeta, "version", ts, PackInt64(expectedVersion+1))
	mut.Set(CFMeta, "updated_at", ts, RFC3339(time.Now()))

	// 3. Build condition filter: exact match on current version.
	cond := bigtable.ChainFilters(
		bigtable.FamilyFilter(CFMeta),
		bigtable.ColumnFilter("version"),
		bigtable.ValueRangeFilter(PackInt64(expectedVersion), PackInt64(expectedVersion+1)),
	)

	// 4. Atomic check-and-mutate.
	matched, err := r.table.CheckAndMutateRow(ctx, AssetKey(assetID), cond, mut, nil)
	if err != nil {
		return 0, fmt.Errorf("AssetRepo.MergeCfAlgo: check-and-mutate: %w", err)
	}
	if !matched {
		return 0, repository.ErrOptimisticLock
	}
	return expectedVersion + 1, nil
}

// ListByMcapFile returns all assets for a given MCAP file via the secondary index.
func (r *AssetRepo) ListByMcapFile(ctx context.Context, mcapFileID string) ([]*models.Asset, error) {
	prefix := mcapFileID + "#"
	var assetIDs []string

	if err := r.idxTable.ReadRows(ctx, bigtable.PrefixRange(prefix), func(row bigtable.Row) bool {
		// row key: <mcap_file_id>#<start_timestamp_ns>#<asset_id>
		if assetID := ParseIdxSegmentAssetID(row.Key()); assetID != "" {
			assetIDs = append(assetIDs, assetID)
		}
		return true
	}, bigtable.RowFilter(bigtable.LatestNFilter(1))); err != nil {
		return nil, fmt.Errorf("AssetRepo.ListByMcapFile: read index: %w", err)
	}

	return r.GetBatch(ctx, assetIDs)
}

// GetBatch fetches multiple assets in a single RPC.
func (r *AssetRepo) GetBatch(ctx context.Context, assetIDs []string) ([]*models.Asset, error) {
	if len(assetIDs) == 0 {
		return nil, nil
	}

	rs := bigtable.RowList{}
	for _, id := range assetIDs {
		rs = append(rs, AssetKey(id))
	}

	var assets []*models.Asset
	if err := r.table.ReadRows(ctx, rs, func(row bigtable.Row) bool {
		assets = append(assets, rowToAsset(row))
		return true
	}, bigtable.RowFilter(bigtable.LatestNFilter(1))); err != nil {
		return nil, fmt.Errorf("AssetRepo.GetBatch: %w", err)
	}
	return assets, nil
}

// McapFileRepo provides read/write access to the `mcap_files` Bigtable table.
type McapFileRepo struct {
	table btTable
}

var _ repository.McapFileRepository = (*McapFileRepo)(nil)

func NewMcapFileRepo(c *Client) *McapFileRepo {
	return &McapFileRepo{table: c.Table(TableMcapFiles)}
}

// Get fetches a single McapFile by ID.
func (r *McapFileRepo) Get(ctx context.Context, mcapFileID string) (*models.McapFile, error) {
	row, err := r.table.ReadRow(ctx, McapFileKey(mcapFileID),
		bigtable.RowFilter(bigtable.LatestNFilter(1)),
	)
	if err != nil {
		return nil, fmt.Errorf("McapFileRepo.Get: %w", err)
	}
	if row == nil {
		return nil, nil
	}
	return rowToMcapFile(row), nil
}

// Set writes all columns of a McapFile.
func (r *McapFileRepo) Set(ctx context.Context, f *models.McapFile) error {
	now := time.Now()
	if f.CreatedAt.IsZero() {
		f.CreatedAt = now
	}
	f.UpdatedAt = now
	f.Version++

	ts := bigtable.Now()
	mut := bigtable.NewMutation()

	// Keep both qualifiers during transition:
	// - mcap_uri (canonical in sql.md)
	// - gcs_path (legacy API field)
	mut.Set(CFMeta, "mcap_uri", ts, B(f.GCSPath))
	mut.Set(CFMeta, "gcs_path", ts, B(f.GCSPath))
	mut.Set(CFMeta, "size_bytes", ts, PackInt64(f.SizeBytes))
	mut.Set(CFMeta, "raw_hash_md5", ts, B(f.RawHashMD5))
	mut.Set(CFMeta, "ingest_state", ts, B(string(f.IngestState)))
	mut.Set(CFMeta, "start_timestamp_ns", ts, PackInt64(f.StartTimestampNs))
	mut.Set(CFMeta, "end_timestamp_ns", ts, PackInt64(f.EndTimestampNs))
	mut.Set(CFMeta, "channel_count", ts, PackInt64(int64(f.ChannelCount)))
	mut.Set(CFMeta, "chunk_count", ts, PackInt64(int64(f.ChunkCount)))
	mut.Set(CFMeta, "owner", ts, B(f.Owner))
	mut.Set(CFMeta, "created_at", ts, RFC3339(f.CreatedAt))
	mut.Set(CFMeta, "updated_at", ts, RFC3339(f.UpdatedAt))
	mut.Set(CFMeta, "version", ts, PackInt64(f.Version))

	for k, v := range f.ProcessState {
		mut.Set(CFProcess, k, ts, B(v))
	}

	if err := r.table.Apply(ctx, McapFileKey(f.McapFileID), mut); err != nil {
		return fmt.Errorf("McapFileRepo.Set: %w", err)
	}
	return nil
}

// List scans mcap_files with pagination and early termination.
// ingestState and owner provide optional client-side filtering (Bigtable fallback mode).
func (r *McapFileRepo) List(ctx context.Context, page, pageSize int, ingestState, owner string) ([]*models.McapFile, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	needed := page * pageSize
	maxScan := needed * 5
	if maxScan < 500 {
		maxScan = 500
	}
	if maxScan > 50000 {
		maxScan = 50000
	}

	scanCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var all []*models.McapFile
	scanned := 0

	scanErr := r.table.ReadRows(scanCtx, bigtable.PrefixRange(RowKeyPrefix), func(row bigtable.Row) bool {
		scanned++
		all = append(all, rowToMcapFile(row))
		return scanned < maxScan
	}, bigtable.RowFilter(bigtable.LatestNFilter(1)))

	if scanErr != nil && scanCtx.Err() == nil {
		return nil, 0, fmt.Errorf("McapFileRepo.List: %w", scanErr)
	}

	// Client-side filtering for Bigtable degraded mode
	if ingestState != "" || owner != "" {
		filtered := make([]*models.McapFile, 0, len(all))
		ownerLower := strings.ToLower(owner)
		for _, f := range all {
			if ingestState != "" && string(f.IngestState) != ingestState {
				continue
			}
			if owner != "" && !strings.Contains(strings.ToLower(f.Owner), ownerLower) {
				continue
			}
			filtered = append(filtered, f)
		}
		all = filtered
	}

	total := int64(len(all))

	// Sort by updated_at DESC.
	sort.Slice(all, func(i, j int) bool {
		return all[i].UpdatedAt.After(all[j].UpdatedAt)
	})

	start := (page - 1) * pageSize
	if start >= len(all) {
		return []*models.McapFile{}, total, nil
	}
	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}
	return all[start:end], total, nil
}

// UpdateIngestState flips the ingest_state field.
func (r *McapFileRepo) UpdateIngestState(ctx context.Context, mcapFileID string, state models.IngestState) error {
	mut := bigtable.NewMutation()
	mut.Set(CFMeta, "ingest_state", bigtable.Now(), B(string(state)))
	mut.Set(CFMeta, "updated_at", bigtable.Now(), RFC3339(time.Now()))
	if err := r.table.Apply(ctx, McapFileKey(mcapFileID), mut); err != nil {
		return fmt.Errorf("McapFileRepo.UpdateIngestState: %w", err)
	}
	return nil
}

// DeliveryRepo provides read/write access to deliveries + secondary indexes.
type DeliveryRepo struct {
	table       btTable
	idxAsset    btTable
	idxCustomer btTable
}

var _ repository.DeliveryRepository = (*DeliveryRepo)(nil)

func NewDeliveryRepo(c *Client) *DeliveryRepo {
	return &DeliveryRepo{
		table:       c.Table(TableDeliveries),
		idxAsset:    c.Table(TableIdxAssetDeliveries),
		idxCustomer: c.Table(TableIdxCustomerDeliveries),
	}
}

// Get fetches a single delivery by ID.
func (r *DeliveryRepo) Get(ctx context.Context, deliveryID string) (*models.Delivery, error) {
	row, err := r.table.ReadRow(ctx, DeliveryKey(deliveryID),
		bigtable.RowFilter(bigtable.LatestNFilter(1)),
	)
	if err != nil {
		return nil, fmt.Errorf("DeliveryRepo.Get: %w", err)
	}
	if row == nil {
		return nil, nil
	}
	return rowToDelivery(row), nil
}

// Set writes all columns of a delivery.
func (r *DeliveryRepo) Set(ctx context.Context, d *models.Delivery) error {
	now := time.Now()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	d.UpdatedAt = now
	d.Version++

	ts := bigtable.Now()
	mut := bigtable.NewMutation()

	mut.Set(CFMeta, "customer_id", ts, B(d.CustomerID))
	mut.Set(CFMeta, "status", ts, B(string(d.Status)))
	mut.Set(CFMeta, "manifest_uri", ts, B(d.ManifestURI))
	mut.Set(CFMeta, "contract_id", ts, B(d.ContractID))
	mut.Set(CFMeta, "note", ts, B(d.Note))
	mut.Set(CFMeta, "asset_count", ts, PackInt64(int64(d.AssetCount)))
	mut.Set(CFMeta, "owner", ts, B(d.Owner))
	mut.Set(CFMeta, "created_at", ts, RFC3339(d.CreatedAt))
	mut.Set(CFMeta, "updated_at", ts, RFC3339(d.UpdatedAt))
	mut.Set(CFMeta, "version", ts, PackInt64(d.Version))
	if d.DeliveredAt != nil {
		mut.Set(CFMeta, "delivered_at", ts, RFC3339(*d.DeliveredAt))
	}

	if err := r.table.Apply(ctx, DeliveryKey(d.DeliveryID), mut); err != nil {
		return fmt.Errorf("DeliveryRepo.Set: %w", err)
	}
	return nil
}

// WriteIndexes writes both secondary index rows for one (asset, delivery) pair.
func (r *DeliveryRepo) WriteIndexes(ctx context.Context, assetID string, d *models.Delivery) error {
	at := d.CreatedAt
	if d.DeliveredAt != nil {
		at = *d.DeliveredAt
	}

	refVal := B(d.DeliveryID) // index row just stores delivery_id as the value

	// idx_asset_deliveries: asset → deliveries
	mutA := bigtable.NewMutation()
	mutA.Set(CFRef, "delivery_id", bigtable.Now(), refVal)
	if err := r.idxAsset.Apply(ctx, IdxAssetDeliveryKey(assetID, at, d.DeliveryID), mutA); err != nil {
		return fmt.Errorf("DeliveryRepo.WriteIndexes: asset idx: %w", err)
	}

	// idx_customer_deliveries: customer → deliveries
	mutC := bigtable.NewMutation()
	mutC.Set(CFRef, "delivery_id", bigtable.Now(), refVal)
	if err := r.idxCustomer.Apply(ctx, IdxCustomerDeliveryKey(d.CustomerID, at, d.DeliveryID), mutC); err != nil {
		return fmt.Errorf("DeliveryRepo.WriteIndexes: customer idx: %w", err)
	}

	return nil
}

// List returns a paginated list of deliveries. Bigtable implementation returns
// an empty list as a degraded fallback — full listing is only supported on Postgres.
func (r *DeliveryRepo) List(ctx context.Context, page, pageSize int, status string) ([]*models.Delivery, int64, error) {
	return []*models.Delivery{}, 0, nil
}

// ListByAsset returns all delivery IDs for a given asset (via secondary index).
func (r *DeliveryRepo) ListByAsset(ctx context.Context, assetID string) ([]string, error) {
	prefix := assetID + "#"
	var ids []string
	if err := r.idxAsset.ReadRows(ctx, bigtable.PrefixRange(prefix), func(row bigtable.Row) bool {
		for _, col := range row[CFRef] {
			ids = append(ids, S(col.Value))
		}
		return true
	}, bigtable.RowFilter(bigtable.LatestNFilter(1))); err != nil {
		return nil, fmt.Errorf("DeliveryRepo.ListByAsset: %w", err)
	}
	return ids, nil
}

// ListItems is not supported on the deprecated Bigtable runtime path.
// It returns an empty slice so legacy tests continue to compile.
func (r *DeliveryRepo) ListItems(context.Context, string) ([]*models.DeliveryItem, error) {
	return []*models.DeliveryItem{}, nil
}

// ListByCustomer returns all delivery IDs for a given customer (via secondary index).
func (r *DeliveryRepo) ListByCustomer(ctx context.Context, customerID string) ([]string, error) {
	prefix := customerID + "#"
	var ids []string
	if err := r.idxCustomer.ReadRows(ctx, bigtable.PrefixRange(prefix), func(row bigtable.Row) bool {
		for _, col := range row[CFRef] {
			ids = append(ids, S(col.Value))
		}
		return true
	}, bigtable.RowFilter(bigtable.LatestNFilter(1))); err != nil {
		return nil, fmt.Errorf("DeliveryRepo.ListByCustomer: %w", err)
	}
	return ids, nil
}

type IdempotencyRepo struct {
	table btTable
}

var _ repository.IdempotencyRepository = (*IdempotencyRepo)(nil)

func NewIdempotencyRepo(c *Client) *IdempotencyRepo {
	return &IdempotencyRepo{table: c.Table(TableIdempotencyKeys)}
}

func idemRowKey(scope, key string) string {
	return RowKeyPrefix + scope + "#" + key
}

// Get returns stored record if present; nil,nil when not found.
func (r *IdempotencyRepo) Get(ctx context.Context, scope, key string) (*repository.IdempotencyRecord, error) {
	row, err := r.table.ReadRow(ctx, idemRowKey(scope, key), bigtable.RowFilter(bigtable.LatestNFilter(1)))
	if err != nil {
		return nil, fmt.Errorf("IdempotencyRepo.Get: %w", err)
	}
	if row == nil {
		return nil, nil
	}
	rec := &repository.IdempotencyRecord{Scope: scope, Key: key}
	for _, col := range row[CFIdem] {
		qual := col.Column[len(CFIdem)+1:]
		switch qual {
		case "request_hash":
			rec.RequestHash = S(col.Value)
		case "status_code":
			v, _ := strconv.Atoi(S(col.Value))
			rec.StatusCode = v
		case "response_json":
			rec.Response = append([]byte(nil), col.Value...)
		case "created_at":
			rec.CreatedAt = ParseRFC3339(col.Value)
		}
	}
	return rec, nil
}

func (r *IdempotencyRepo) Save(ctx context.Context, rec *repository.IdempotencyRecord) error {
	mut := bigtable.NewMutation()
	ts := bigtable.Now()
	mut.Set(CFIdem, "request_hash", ts, B(rec.RequestHash))
	mut.Set(CFIdem, "status_code", ts, B(strconv.Itoa(rec.StatusCode)))
	mut.Set(CFIdem, "response_json", ts, rec.Response)
	mut.Set(CFIdem, "created_at", ts, RFC3339(time.Now()))
	if err := r.table.Apply(ctx, idemRowKey(rec.Scope, rec.Key), mut); err != nil {
		return fmt.Errorf("IdempotencyRepo.Save: %w", err)
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// AlgoEventRepo
// ──────────────────────────────────────────────────────────────────────────────

// AlgoEventRepo provides read/write access to the `asset_algo_events` Bigtable table.
type AlgoEventRepo struct {
	table btTable
}

// Ensure Bigtable implementation satisfies repository abstraction.
var _ repository.AlgoEventRepository = (*AlgoEventRepo)(nil)

func NewAlgoEventRepo(c *Client) *AlgoEventRepo {
	return &AlgoEventRepo{table: c.Table(TableAssetAlgoEvents)}
}

// algoEventRowKey builds the row key: <asset_id>#<reverse_timestamp_20digits>#<event_id>.
// reverse_timestamp ensures newest events sort first in lexicographic order.
func algoEventRowKey(event *models.AlgoEvent) string {
	rev := math.MaxInt64 - event.CreatedAt.UnixMilli()
	return fmt.Sprintf("%s#%020d#%s", event.AssetID, rev, event.EventID)
}

// Insert persists a single algorithm status-change event.
func (r *AlgoEventRepo) Insert(ctx context.Context, event *models.AlgoEvent) error {
	rowKey := algoEventRowKey(event)
	mut := bigtable.NewMutation()
	ts := bigtable.Now()

	mut.Set(CFMeta, "event_id", ts, B(event.EventID))
	mut.Set(CFMeta, "asset_id", ts, B(event.AssetID))
	mut.Set(CFMeta, "algo_key", ts, B(event.AlgoKey))
	if event.PrevStatus != nil {
		mut.Set(CFMeta, "prev_status", ts, B(*event.PrevStatus))
	}
	mut.Set(CFMeta, "new_status", ts, B(event.NewStatus))
	if event.RunID != nil {
		mut.Set(CFMeta, "run_id", ts, B(*event.RunID))
	}
	if event.Reason != nil {
		mut.Set(CFMeta, "reason", ts, B(*event.Reason))
	}
	mut.Set(CFMeta, "created_at", ts, RFC3339(event.CreatedAt))

	if err := r.table.Apply(ctx, rowKey, mut); err != nil {
		return fmt.Errorf("AlgoEventRepo.Insert: %w", err)
	}
	return nil
}

// ListByAsset returns all events for the given asset, ordered by created_at DESC.
// When algoKey is non-nil, only events matching that algo_key are returned.
func (r *AlgoEventRepo) ListByAsset(ctx context.Context, assetID string, algoKey *string) ([]*models.AlgoEvent, error) {
	prefix := assetID + "#"
	var events []*models.AlgoEvent

	if err := r.table.ReadRows(ctx, bigtable.PrefixRange(prefix), func(row bigtable.Row) bool {
		ev := rowToAlgoEvent(row)
		if algoKey != nil && ev.AlgoKey != *algoKey {
			return true // skip non-matching, continue scanning
		}
		events = append(events, ev)
		return true
	}, bigtable.RowFilter(bigtable.LatestNFilter(1))); err != nil {
		return nil, fmt.Errorf("AlgoEventRepo.ListByAsset: %w", err)
	}
	return events, nil
}

// rowToAlgoEvent converts a Bigtable row to an AlgoEvent model.
func rowToAlgoEvent(row bigtable.Row) *models.AlgoEvent {
	ev := &models.AlgoEvent{}
	for _, col := range row[CFMeta] {
		qual := col.Column[len(CFMeta)+1:]
		switch qual {
		case "event_id":
			ev.EventID = S(col.Value)
		case "asset_id":
			ev.AssetID = S(col.Value)
		case "algo_key":
			ev.AlgoKey = S(col.Value)
		case "prev_status":
			s := S(col.Value)
			ev.PrevStatus = &s
		case "new_status":
			ev.NewStatus = S(col.Value)
		case "run_id":
			s := S(col.Value)
			ev.RunID = &s
		case "reason":
			s := S(col.Value)
			ev.Reason = &s
		case "created_at":
			ev.CreatedAt = ParseRFC3339(col.Value)
		}
	}
	return ev
}

// Row → model conversion helpers.
func rowToAsset(row bigtable.Row) *models.Asset {
	a := &models.Asset{
		AssetID:       rowKeyID(row.Key()),
		AlgoResults:   map[string]string{},
		Tags:          map[string]string{},
		Files:         map[string]string{},
		LifecycleMeta: map[string]interface{}{},
	}

	for _, col := range row[CFMeta] {
		qual := col.Column[len(CFMeta)+1:] // strip "meta:" prefix
		switch qual {
		case "mcap_file_id":
			a.McapFileID = S(col.Value)
		case "start_timestamp_ns":
			a.StartTimestampNs = UnpackInt64(col.Value)
		case "end_timestamp_ns":
			a.EndTimestampNs = UnpackInt64(col.Value)
		case "duration_sec":
			a.DurationSec, _ = strconv.ParseFloat(S(col.Value), 64)
		case "reviewer":
			a.Reviewer = S(col.Value)
		case "status":
			a.Status = models.AssetStatus(S(col.Value))
		case "owner":
			a.Owner = S(col.Value)
		case "type":
			a.SegType = S(col.Value)
		case "env":
			a.Env = S(col.Value)
		case "task":
			a.Task = S(col.Value)
		case "delivery_count":
			a.DeliveryCount = int(UnpackInt64(col.Value))
		case "last_delivered_to":
			a.LastDeliveredTo = S(col.Value)
		case "last_delivered_at":
			t := ParseRFC3339(col.Value)
			a.LastDeliveredAt = &t
		case "created_at":
			a.CreatedAt = ParseRFC3339(col.Value)
		case "updated_at":
			a.UpdatedAt = ParseRFC3339(col.Value)
		case "version":
			a.Version = UnpackInt64(col.Value)
		case "retention_tier", "archive_after_days", "delete_after_days", "total_size_bytes", "last_accessed_at":
			a.LifecycleMeta[qual] = S(col.Value)
		}
	}

	for _, col := range row[CFAlgo] {
		qual := col.Column[len(CFAlgo)+1:]
		a.AlgoResults[qual] = S(col.Value)
	}

	for _, col := range row[CFTag] {
		qual := col.Column[len(CFTag)+1:]
		a.Tags[qual] = S(col.Value)
	}

	for _, col := range row[CFFiles] {
		qual := col.Column[len(CFFiles)+1:]
		a.Files[qual] = S(col.Value)
	}

	return a
}

func rowToMcapFile(row bigtable.Row) *models.McapFile {
	f := &models.McapFile{
		McapFileID:   rowKeyID(row.Key()),
		ProcessState: map[string]string{},
	}

	for _, col := range row[CFMeta] {
		qual := col.Column[len(CFMeta)+1:]
		switch qual {
		case "mcap_uri":
			f.GCSPath = S(col.Value)
		case "gcs_path":
			// Backward compatibility: only apply if canonical field is absent.
			if f.GCSPath == "" {
				f.GCSPath = S(col.Value)
			}
		case "size_bytes":
			f.SizeBytes = UnpackInt64(col.Value)
		case "raw_hash_md5":
			f.RawHashMD5 = S(col.Value)
		case "ingest_state":
			f.IngestState = models.IngestState(S(col.Value))
		case "start_timestamp_ns":
			f.StartTimestampNs = UnpackInt64(col.Value)
		case "end_timestamp_ns":
			f.EndTimestampNs = UnpackInt64(col.Value)
		case "channel_count":
			f.ChannelCount = int(UnpackInt64(col.Value))
		case "chunk_count":
			f.ChunkCount = int(UnpackInt64(col.Value))
		case "owner":
			f.Owner = S(col.Value)
		case "created_at":
			f.CreatedAt = ParseRFC3339(col.Value)
		case "updated_at":
			f.UpdatedAt = ParseRFC3339(col.Value)
		case "version":
			f.Version = UnpackInt64(col.Value)
		}
	}

	for _, col := range row[CFProcess] {
		qual := col.Column[len(CFProcess)+1:]
		f.ProcessState[qual] = strconv.Quote(S(col.Value))
	}

	return f
}

func rowToDelivery(row bigtable.Row) *models.Delivery {
	d := &models.Delivery{DeliveryID: rowKeyID(row.Key())}
	for _, col := range row[CFMeta] {
		qual := col.Column[len(CFMeta)+1:]
		switch qual {
		case "customer_id":
			d.CustomerID = S(col.Value)
		case "status":
			d.Status = models.DeliveryStatus(S(col.Value))
		case "manifest_uri":
			d.ManifestURI = S(col.Value)
		case "contract_id":
			d.ContractID = S(col.Value)
		case "note":
			d.Note = S(col.Value)
		case "asset_count":
			d.AssetCount = int(UnpackInt64(col.Value))
		case "owner":
			d.Owner = S(col.Value)
		case "delivered_at":
			t := ParseRFC3339(col.Value)
			d.DeliveredAt = &t
		case "created_at":
			d.CreatedAt = ParseRFC3339(col.Value)
		case "updated_at":
			d.UpdatedAt = ParseRFC3339(col.Value)
		case "version":
			d.Version = UnpackInt64(col.Value)
		}
	}
	return d
}

// rowKeyID strips the "v1#" prefix from a row key.
func rowKeyID(key string) string {
	if len(key) > 3 {
		return key[3:]
	}
	return key
}
