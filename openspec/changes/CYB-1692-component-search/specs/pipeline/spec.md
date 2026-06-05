## ADDED Requirements

### Requirement: Component palette supports local search in the pipeline designer

The pipeline designer MUST allow users to search the left-side component palette without leaving the page.

#### Scenario: Filtering components by search term

- **Given** the pipeline designer has loaded reusable components
- **When** the user types a search term into the component palette search input
- **Then** the palette shows only components whose name, image, tag, or compute tier matches the term case-insensitively

#### Scenario: Clearing the search restores the full palette

- **Given** the component palette is filtered by a search term
- **When** the user clears the search input
- **Then** the full component list is shown again

#### Scenario: No matching components

- **Given** the component palette has loaded components
- **When** the search term matches none of them
- **Then** the palette shows a search-specific empty state instead of the normal component list

#### Scenario: Dragging a filtered component still works

- **Given** the component palette is filtered
- **When** the user drags one of the visible components into the canvas
- **Then** the designer creates the same node payload it would have created without filtering
