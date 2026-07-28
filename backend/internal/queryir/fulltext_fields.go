package queryir

// FulltextExtraFields lists the flattened, top-level asset columns that the
// free-text ("top search box") path must match in addition to the built-in
// core fields (asset_id / notes / owner / reviewer / …).
//
// These are the flatten-mirror columns already written onto the ES document by
// searchindex.Builder and stored as real columns on the assets table, so they
// require no reindex to become queryable — only the query builders needed to
// start searching them. CYB-4011.
//
// SINGLE SOURCE OF TRUTH: both the Elasticsearch keyword query
// (elasticsearch.buildSearchModeQuery) and the PostgreSQL fulltext fallback
// (queryexec/postgres.buildFulltextClause) iterate this slice. To make another
// already-flattened field top-searchable, add it here — do NOT edit the two
// builders independently, or ES and PG search behavior will drift.
//
// Only add fields that are (a) real columns on the assets table AND (b) written
// to the ES doc by the Builder. A genuinely new field additionally needs the
// Builder to emit it and old docs reindexed before it will match in ES.
var FulltextExtraFields = []string{
	"grace_video_id", // CYB-4011: Grace video UUID
	"device_id",      // CYB-4011: collection device identifier
}
