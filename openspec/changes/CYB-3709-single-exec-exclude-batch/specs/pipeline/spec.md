## MODIFIED Requirements

### Requirement: 单次执行列表只显示单次 run,不含批量任务

The single-execution list (the non-batch scope of the executions view) SHALL show only runs that are not part of a batch. Both batch parent (aggregate) rows and batch child rows SHALL be excluded from this list. Batch children SHALL remain listed in the batch detail view (filtered by batch job id), which is unchanged.

#### Scenario: batch children do not appear in the single-execution list

- **GIVEN** batch jobs whose child runs each carry a batch job id
- **WHEN** the user opens the single-execution list (non-batch scope)
- **THEN** the list SHALL contain only runs with no batch association
- **AND** it SHALL NOT contain batch parent rows or batch child rows

#### Scenario: batch children remain visible in the batch view

- **GIVEN** a batch job with child runs
- **WHEN** the user opens that batch's detail view
- **THEN** the batch's child runs SHALL be listed as before
