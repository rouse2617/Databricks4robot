## Overview

This change adds lightweight client-side search to the component palette used by the pipeline designer. The current palette already loads the full component list in memory, so filtering can be done locally without changing backend APIs.

## Design

### Search input

- Add a compact search input at the top of `ComponentPalette`
- Placeholder should clearly indicate component search
- Search state lives inside the palette unless a higher-level parent state is required by existing tests

### Filter behavior

- Case-insensitive match across:
  - component name
  - image
  - tag
  - compute tier
- Trim whitespace before matching
- Empty search term returns the unfiltered list

### States

- Loading: unchanged
- Error: unchanged
- No components loaded at all: existing empty state remains
- No matches for current search: show a search-specific empty state with a clear reset path

### Drag-and-drop

- Filtering must not change the drag payload shape
- Filtered items must still create the same pipeline nodes as before

## Non-goals

- No backend/server-side search
- No grouping, favorites, or recent components
- No changes to component CRUD screens
