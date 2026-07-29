# CYB-4427 decisions

## Wrapper over backend-dev.sh, not a forked prod script
backend-dev.sh is already fully `${VAR:-default}`-parameterized. A thin wrapper
that exports prod values and `exec`s it means ONE deploy implementation; a
forked backend-prod.sh would drift. Cost: four small additive hooks in the
shared script. All default to no-op, so dev is byte-for-byte unchanged
(verified: deploy-dev.yml sets none of the switched vars).

## `${V:-d}` → `${V-d}` for five vars (why it's dev-safe)
`:-` treats an explicit empty string as "use default", so prod couldn't disable
Grace / dev bearer-token / dev webhook-token by exporting empty — the DRY_RUN
showed prod wrongly inheriting the dev Grace URL and binding dev secrets. `-`
(unset→default, empty→empty) fixes it. Safe because dev never sets these
(unset → still default). This was caught only because DRY_RUN was built first.

## Grace sync disabled on prod
The live prod service has no `GRACE_*` — there is no prod Grace endpoint, and
pointing prod at the dev Grace URL would be wrong. Disabled via empty overrides.

## JWT_SECRET / ADMIN_TOKEN kept as plaintext env (for now)
Both are plaintext on the live service. Committing them to git is unacceptable,
and minting Secret Manager entries is a prod action out of this PR's scope. They
survive redeploys via `preserve_all_cloudrun_env_vars`. Flagged as a SECURITY
TODO in backend-prod.sh with the exact EXTRA_SECRET_MAPPINGS line to use once
`cyber-databrew-prod-jwt-secret` / `-admin-token` secrets exist (and ADMIN_TOKEN
is rotated off the throwaway "tmp-reindex-001").

## Prod parity ADDS dev vars prod lacked
FRONTEND_BASE_URL, PIPELINE_RESOURCE_MAX_*, ARGO_WORKFLOW_TTL…, watcher interval,
pricing path — all benign config dev has and prod didn't. Adding them is the
point of "parity". Called out in the PR so the first prod deploy's diff is
expected, not a surprise.

## No app overlay for prod (overlays/prod skipped)
`overlays/dev` patches the `base/` K8s **Deployment** — that path runs the
backend as an in-cluster pod. Prod runs on Cloud Run (via backend-prod.sh), so a
prod app overlay would codify a Deployment prod never uses. The meaningful IaC
is `prod-deps/` (the ES dependency), which is what was actually missing.

## Prod resources left at 1 vCPU / 512Mi
Matches the live service. Bumping toward dev's 4/4Gi (the backend runs the
submitter + watcher + ES subscriber) is a cost/behavior decision deferred to a
follow-up, not silently changed here.
