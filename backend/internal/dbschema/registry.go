package dbschema

// AllModels returns every GORM model that should be registered as the
// schema source-of-truth for Atlas `migrate diff`. The set covers all 47
// tables across 28 SQL migrations (000_initial + 039-064).
//
// When adding a new model:
//   1. Write the struct with full gorm tags (column, type, nullability,
//      default). CHECK constraints and indexes are NOT expressible in
//      GORM tags and should remain in the SQL migration.
//   2. Add a TableName() method returning the SQL table name.
//   3. Append the pointer here in alphabetical order (or by SQL file
//      order — pick one and stay consistent).
//
// When modifying a model:
//   - Edit the tags to match the intended new schema.
//   - Run `atlas migrate diff <name> --env gorm` to auto-generate the
//     new SQL migration.
//   - Commit the generated SQL + the model change together.
func AllModels() []any {
	return []any{
		// 000_initial.sql (assets / asset events / actions / algo runs / etc.)
		&Action{},
		&AlgoRun{},
		&Asset{},
		&AssetAlgoLatest{},
		&AssetEvalResult{},
		// &AssetEvent{} and &AssetEventDefault{} — excluded: GORM doesn't
		// model Postgres table partitioning. The `asset_events` parent +
		// monthly partitions are managed entirely by SQL migrations.
		&AssetRelation{},
		&AssetTag{},
		&AssetUsageStat{},
		&AuditEvent{},
		&Customer{},
		&Delivery{},
		&DeliveryItem{},
		&DeliveryRule{},
		&ESSyncCheckpoint{},
		&IdempotencyKey{},
		&LakehouseBronzeCheckpoint{},
		&LogicalAsset{},
		&McapFile{},
		&OutboxDLQ{},
		&SavedQuery{},
		&SearchReindexJob{},
		&SyncWatermark{},

		// 039-064: pipeline / backfill / run kernel
		&BackfillItem{},
		&BackfillJob{},
		&ComponentBuildRun{},
		&ComponentRelease{},
		&DatabrewRun{},
		&ExecutionTarget{},
		&PipelineComponent{},
		&PipelineComponentRelease{},
		&PipelineConfig{},
		&PipelineConfigVersion{},
		&PipelineDeployment{},
		&PipelineRun{},
		&PipelineRunAssetNode{},
		&PipelineRunEvent{},
		&PipelineRunNode{},
		&PipelineRunNotificationCandidate{},
		&PipelineRunWatcherState{},
		&PipelineTemplate{},
		&RagBuildRun{},
		&RunInput{},
		&RunRelation{},
		&VideoDuration{},
	}
}
