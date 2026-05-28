-- Asset model expansion Phase 1: allow dataset and annotation lineage edges.

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
      'materialized_from'
    )
  );

ALTER TABLE assets
  DROP CONSTRAINT IF EXISTS chk_mcap_file_required,
  ADD CONSTRAINT chk_mcap_file_required CHECK (
    asset_type IN ('derived_asset', 'dataset', 'annotation_result') OR mcap_file_id IS NOT NULL
  );
