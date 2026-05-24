## ADDED Requirements

### Requirement: Asset detail delivery history uses bounded asset scope

Asset detail delivery history SHALL load delivery rows for the selected asset through an asset-scoped or otherwise bounded query and SHALL settle to either rows or an empty state without scanning unbounded global delivery pages.

#### Scenario: Asset has delivery history

- **GIVEN** an asset detail page is opened for an asset with known delivery history
- **WHEN** the user selects the delivery history tab
- **THEN** the UI displays the asset's delivery rows
- **AND** the UI does not repeatedly request unrelated global delivery pages
- **AND** the loading indicator clears after the bounded request completes

#### Scenario: Asset has no delivery history

- **GIVEN** an asset detail page is opened for an asset without delivery rows
- **WHEN** the user selects the delivery history tab
- **THEN** the UI displays the empty delivery-history state
- **AND** the loading indicator clears after the bounded request completes

### Requirement: Asset discovery reset and delivery summary copy keep state consistent

Asset discovery and detail surfaces SHALL keep visible input state aligned with applied query state, and SHALL label delivery counters by their actual counting semantics.

#### Scenario: User clears a no-results asset query

- **GIVEN** an asset search query returns no results
- **WHEN** the user clicks "放宽筛选" or "回到默认视图"
- **THEN** the applied filters are cleared
- **AND** the search input text is cleared
- **AND** the default asset list is shown

#### Scenario: Asset has cancelled delivery history but no completed deliveries

- **GIVEN** an asset detail summary has `delivery_count` of 0
- **AND** its delivery history includes a cancelled delivery
- **WHEN** the user views the asset summary and delivery history tab
- **THEN** the summary labels the counter as completed deliveries
- **AND** it does not imply that total delivery history is empty
