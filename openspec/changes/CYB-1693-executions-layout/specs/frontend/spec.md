# Frontend Spec Delta

## Modified Behavior

### Pipeline execution list remains within the page viewport

The Pipeline execution records tab MUST keep its header, filters, and table within the visible app content area on desktop screens.

#### Scenario: table is wider than viewport
- **Given** a desktop user opens the Pipeline execution records tab
- **When** the execution table has many columns
- **Then** the page content is not clipped on the left edge
- **And** horizontal overflow is contained inside the table region
- **And** the operation column remains reachable

#### Scenario: no rows selected for bulk delete
- **Given** no execution rows are selected
- **When** the bulk delete button is disabled
- **Then** the UI explains that rows must be selected first

#### Scenario: failed execution row
- **Given** an execution row has Failed or Error status
- **When** the user scans the row
- **Then** the row provides a clear detail entry for diagnosing the failure
