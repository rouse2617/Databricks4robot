-- CYB-1220: Add CHECK constraint for asset_relations.relation_type
--
-- 6 valid edge types defined in the unified asset hierarchy:
--   split_from, derived_from, contains, sampled_from, merged_from, revision_of
--
-- ⚠️ Off-limits zone — approved by user.
-- See openspec/changes/CYB-1220-asset-relations-check/

ALTER TABLE asset_relations
  DROP CONSTRAINT IF EXISTS chk_relation_type,
  ADD CONSTRAINT chk_relation_type CHECK (
    relation_type IN ('split_from','derived_from','contains','sampled_from','merged_from','revision_of')
  );
