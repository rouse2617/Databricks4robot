## ADDED Requirements

### Requirement: Documentation crawler metadata
The system SHALL serve valid crawler metadata files from the documentation site and root site instead of falling back to an HTML application shell.

**Priority**: P1 (High)
**Rationale**: Invalid crawler metadata causes Lighthouse SEO failures and makes the public documentation harder for search engines and AI crawlers to interpret.

#### Scenario: robots metadata is available
- **Given** the documentation site is built and served under `/doc/`
- **When** a crawler requests `/doc/robots.txt`
- **Then** the response contains valid robots directives rather than HTML markup

#### Scenario: root robots metadata is available
- **Given** the main frontend site is built and served at the origin root
- **When** a crawler requests `/robots.txt`
- **Then** the response contains valid robots directives rather than HTML markup

#### Scenario: llms metadata is available
- **Given** the documentation site is built and served under `/doc/`
- **When** a crawler requests `/doc/llms.txt`
- **Then** the response contains Markdown with an H1 heading and relevant documentation links

#### Scenario: root llms metadata is available
- **Given** the main frontend site is built and served at the origin root
- **When** a crawler requests `/llms.txt`
- **Then** the response contains Markdown with an H1 heading and relevant documentation links

## MODIFIED Requirements

### Requirement: Documentation navigation clarity
- **Before**: The system SHALL expose the documentation entry with an ambiguous top-level label.
- **After**: The system SHALL expose the documentation entry with a top-level label that describes the linked documentation content.
- **Reason**: Misleading labels make users choose the wrong path and reduce trust in the documentation.

#### Scenario: top navigation labels match destinations
- **Given** a user opens the documentation site
- **When** they inspect the top navigation
- **Then** each visible label describes the page or section it opens

#### Scenario: quick-start paths are discoverable
- **Given** a user opens the overview page for the first time
- **When** they scan the first page section
- **Then** they can immediately choose a quick start, SDK, API, or asset guide path without scrolling to the bottom

### Requirement: Documentation readability
- **Before**: The system SHALL use default documentation styling that can produce low-contrast links and sparse wide-screen layouts.
- **After**: The system SHALL keep documentation text readable with accessible link contrast and a more balanced desktop layout.
- **Reason**: Low contrast harms accessibility, and excessive empty space on wide screens makes the page feel unfinished.

#### Scenario: links remain readable
- **Given** the overview page is rendered in light mode
- **When** accessibility checks inspect article links and breadcrumbs
- **Then** foreground and background colors meet WCAG AA contrast expectations for normal text

#### Scenario: layout remains responsive
- **Given** the overview page is rendered on desktop and mobile viewports
- **When** the user scans and scrolls the page
- **Then** the content has no horizontal overflow or overlapping navigation, headings, links, or body text
