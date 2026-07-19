# Decisions — CYB-3680

- **Owner-less + sliding TTL, not refcounts**: the CM is shared; cascading
  with any single Workflow would strand the rest. Janitor stays dumb (age
  check only); safety = every ensure touches the timestamp + DB blob makes
  deletion recoverable. Review round 2 (P1-2): touch is rate-limited via an
  in-memory per-hash cache shared across per-target stores (~1 PATCH/h/hash).
- **TTL 35d > workflow TTL 30d**: Argo-native retry re-mounts the CM while
  the Workflow object lives; the reference timestamp must outlive it.
- **Blob failures don't block dispatch**: the CM was ensured directly; the
  blob only powers rebuild. Warn + continue.
- **Janitor scope v1 = default cluster** (cluster-wide list by label).
  Per-cluster sweeps ride the per-cluster channel loops (CYB-3678/3681).
- **Dual-rail transition**: legacy `runtime-config-<run-id>` CMs keep owner
  cascade; janitor's selector can never touch them.
