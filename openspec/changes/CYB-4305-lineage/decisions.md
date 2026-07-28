# CYB-4305 decisions

## Why two depth modes instead of always-all

Depth=1 is a single indexed SELECT on `assets` (direct columns from CYB-3268 + CYB-3283). Depth=all requires an ES `_mget` — cheap but not free at 5000 ids. Users doing bulk root-reverse lookup ("which mcap did each segment come from?") only need depth=1; they'd pay the ES round-trip for nothing. Split path keeps the fast path fast.

## Why merge ES + SQL server-side, not client-side

Two round-trips from the frontend would be 2× latency + fragile ordering. Server has both handles (Postgres pool + ES client) and can join once in memory. Also keeps the wire response homogeneous — client doesn't need to know which fields came from where.

## Why `orphan_count` and `is_root_count` as separate stats

They mean different things:
- **is_root** — legitimate top of a lineage tree (mcap, raw upload, etc.)
- **orphan** — parent AND root both null — usually a data-quality bug (asset created without lineage relation being written)

Merging them would hide the diagnostic signal. Users triaging orphans want to see that count in isolation.

## Why `is_root_count` counts both `root == self` and `root IS NULL`

Both encode "no ancestry above me." Historical asset ingestion left some rows with `root_asset_id` NULL when the asset IS its own root (esp. raw_mcap); newer ingestion writes `root_asset_id = asset_id`. Counting both keeps the stat honest across the data-migration boundary.

## Why relation_types is `[]string` not a single enum

Argo lineage relations aren't mutually exclusive: an asset might be a `derive` of one parent AND a `split` of another (rare but real). Array preserves both. Frontend joins with `|` in CSV / `,` in table; server aggregates by union count.

## Why ES `_mget` instead of a single ES query with `terms`

`_mget` returns docs in the request order and preserves missing ids as null hits — matches our stats logic exactly. A `terms` query would return docs in relevance/score order (default), requiring a client-side reindex map. `_mget` is cheaper (no scoring) and semantically correct here.

## Why NOT put lineage in `/queries/run`

`/queries/run` is filter-scan-return over assets; it doesn't project upstream/downstream ids into the item response by default (they're in the ES doc but not surfaced by the query IR). Adding them would grow the IR surface for one caller. A dedicated endpoint keeps the IR lean and this specialized aggregate has a clear contract.

## Why depth type accepts both `1` (int) and `"1"` (string)

Frontend TypeScript sends `depth: 1` naturally (Number); some CLI callers might URL-encode as `"1"`. Server normalizes both to the same code path. `"all"` is only the string. Cheap ergonomic robustness.

## What we're NOT doing

- Recursive multi-hop SQL — projection already flattened the graph at write time (CYB-3268)
- Cycle protection at runtime — projection is DAG-guaranteed at write time
- Server-side lineage graph rendering — single-asset detail page owns that view; batch page is a table
- Live subscription — lineage doesn't change fast enough to warrant streaming
