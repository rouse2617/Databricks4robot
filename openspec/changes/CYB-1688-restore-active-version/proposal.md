# CYB-1688 restore pipeline active version UI

## Why

Pipeline active version support still exists in the backend, but the frontend entry point was removed during later DeployPanel cleanup. Users can no longer choose an older known-good template version as the default run version from the UI.

## What Changes

- Restore version history access from the pipeline template list.
- Restore "set active version" action in the version history drawer.
- Show the active version tag when it differs from the latest version.
- Default the run modal version selector to the active version when one is set.

## Impact

Frontend-only regression fix. No backend API shape changes.
