## Why

The pipeline designer's left component palette does not provide search. As the component registry grows, users have to scroll through the full list to find a step, which slows authoring and makes the designer materially less usable than the component management page.

## What Changes

- Add a search input to the pipeline designer component palette
- Filter palette items client-side by component `name`, `image`, `tag`, and `computeTier`
- Preserve drag-and-drop behavior for filtered results
- Show a search-specific empty state when no components match
- Keep loading and error states unchanged

## Impact

- Frontend-only runtime change under `Frontend/`
- No backend API contract changes
- Affects the `/pipeline` design experience only
