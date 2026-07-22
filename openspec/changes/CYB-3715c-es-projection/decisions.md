# CYB-3715c decisions

## Why not drop the nested `mcap.<col>` object

`mcap.camera_model`, `mcap.device_id`, etc. are already advertised in `docs/review/api-guide.md#filter` and in `filter/assets_fields.go#mcapFilterColumns` (via the `mcap.` prefix). Callers using that path still work post-CYB-3715 because filter/build.go emits an EXISTS subquery over `mcap_files`. Removing the nested projection would break those callers without any migration path. Keeping both is small storage overhead (7 strings × ~800 docs = negligible) for a much smaller blast radius.

## Why omit empty fields instead of always emitting

ES dynamic mapping creates a keyword sub-field on first indexed value. If we always emit even empty strings, every doc gets 7 empty-string keyword buckets in aggregation (`""`: 800), which pollutes facet responses. The existing `mcap.<col>` projection already uses the omit-when-empty pattern (see CYB-3297 Phase C at builder.go:197-224); this PR follows the same convention.

## Why not update the ES mapping template

Deploy uses dynamic mapping (see `deploy/local/elasticsearch/init-index.sh` reference in builder.go docstring). First non-empty indexed value creates a `text` + `.keyword` sub-field, which is what the filter/facet compiler expects (`keywordAggField` appends `.keyword`). A future PR can lock the mapping down explicitly if we want to reject unexpected new fields, but that's out of scope here.

## Why require `admin/search/reindex` for existing docs

The subscriber only writes on `assets.updated_at` change. Existing pangzi raw_mcap rows on dev were written before CYB-3715 landed, so their ES docs pre-date the new columns. Options:
1. **Reindex all docs** — 1 admin call, ~800 docs, done in ~seconds
2. **Bump a version marker** — force subscriber to re-emit — requires touching the subscriber machinery for a one-time backfill (yuck)
3. **UPDATE assets SET updated_at = now()** on affected rows — coarse, might race with in-flight writes

Reindex is the standard tool for this exact case and is idempotent. Tasks.md tracks it as a post-merge action.
