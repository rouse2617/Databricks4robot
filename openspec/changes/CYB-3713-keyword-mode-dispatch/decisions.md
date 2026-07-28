# Decisions — CYB-3713

## 2026-07-21 — Injection at `queryir.Normalize`, not at handler or planner

- **Context**: The synthesized `_fulltext` predicate can be inserted at three layers: (a) the HTTP handler before `Compile`, (b) `queryir.Normalize`, or (c) `queryplan.NewPGBridgePlanner`.
- **Decision**: Do it in `queryir.Normalize`.
- **Alternatives**: Handler layer would work but leaks IR-shape concerns into the HTTP boundary; planner layer would work but the planner is supposed to route already-normalized IR, not mutate the where tree.
- **Rationale**: `Normalize` is the canonical spot for shape rewrites (it already lower-cases mode, trims strings, normalizes sort/pages). Adding the injection here means every consumer (handler, `Validate`, saved queries) inherits the fix for free — including any future planner variants.

## 2026-07-21 — Include `semantic` and `similar` in the same injection

- **Context**: Only `keyword` was in the acceptance test set. But `semantic` and `similar` had the same latent bug — they set `plan.UseESRecall = true` yet lacked a `_fulltext` predicate, so ES received `match_all` for callers who supplied `q` instead of `where`.
- **Decision**: Inject for all three modes when `q != ""`.
- **Alternatives**: Fix `keyword` only, ticket the other two as follow-ups.
- **Rationale**: Symmetric plumbing removes ambiguity for API users and clients. The cost of injecting is one predicate; the cost of a follow-up ticket is another PR + review + deploy cycle. Since `buildSearchModeQuery` already differentiates by mode (`semantic` fuzzy, `similar` MLT, default = keyword non-fuzzy), the mode-specific ES query shape is preserved even when the predicate is uniformly injected.

## 2026-07-21 — OpenSpec written alongside implementation, checkpoint waived

- **Context**: User explicitly delegated the full close-loop for CYB-3713/3714/3715 in one message: "你自己完成,并且闭环测试,我不参与,全程按照开发规范来". `AI-RULES.md` step 4 requires stop-and-confirm on OpenSpec before writing runtime code.
- **Decision**: Write OpenSpec + code without stopping for user approval. The blanket delegation is the approval; user is opted out of the checkpoint by their own explicit instruction.
- **Alternatives**: Pause and wait for confirmation on each ticket — would violate the "我不参与" instruction.
- **Rationale**: `AI-RULES.md#rule-precedence` row 1 (explicit user instruction in the current message) outranks row 5 (OpenSpec traceability). The OpenSpec artifacts are still produced; only the pause-for-confirm is waived.

## 2026-07-21 — Coverage scope is exactly the current fulltext field set

- **Context**: Ticket description mentioned that `mcap_files.metadata` (`vibecap_tasks`, `location.address`, `source_platform`, `collection_session_id`, `collector_height`) is where users want keyword search to reach. But the existing PG `buildFulltextClause` (`expr.go:120`) covers only `asset_id`, `asset_type`, `owner`, `reviewer`, `mcap_file_id`, `notes` tag; ES `buildSearchModeQuery` covers the same set on the ES doc side. Adding metadata JSONB requires either (a) ES mapping change + subscriber projection or (b) PG JSONB expression in `buildFulltextClause`.
- **Decision**: Keep coverage at the current 6-field set. Document explicitly in `spec.md` that metadata is out of scope for this ticket.
- **Alternatives**: Expand the fulltext clause to include `mcap_files.metadata::text ILIKE`. This is 3 lines of code but changes the semantics of every `_fulltext` predicate — worth its own ticket with a reviewer thinking about performance (JSONB casting on every row) and index strategy.
- **Rationale**: The bug is the dispatch, not the coverage. Fixing the dispatch alone gets keyword mode working end-to-end; the coverage expansion is a separate concern tracked by CYB-3714 (tag_registry) and CYB-3715 (column flatten). Attempting all three in one PR would be scope creep against a security-adjacent fix (users see wrong data because search is silently broken).
