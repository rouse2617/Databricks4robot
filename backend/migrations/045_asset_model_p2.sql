-- Asset model expansion Phase 2: ML model and evaluation report asset types.

ALTER TABLE asset_relations
  DROP CONSTRAINT IF EXISTS chk_relation_type,
  ADD CONSTRAINT chk_relation_type CHECK (
    relation_type IN (
      'split_from',
      'derived_from',
      'contains',
      'sampled_from',
      'merged_from',
      'revision_of',
      'annotated_from',
      'materialized_from',
      'trained_from',
      'evaluated_on',
      'validated_on',
      'configured_by',
      'fine_tuned_from',
      'features_from',
      'tested_on',
      'evaluates',
      'compares_to',
      'calibrated_from',
      'generated_by'
    )
  );

ALTER TABLE asset_relations
  ADD COLUMN IF NOT EXISTS metadata jsonb DEFAULT '{}'::jsonb NOT NULL;

ALTER TABLE assets
  DROP CONSTRAINT IF EXISTS chk_mcap_file_required,
  ADD CONSTRAINT chk_mcap_file_required CHECK (
    asset_type IN ('derived_asset', 'dataset', 'annotation_result', 'ml_model', 'evaluation_report')
    OR mcap_file_id IS NOT NULL
  );
