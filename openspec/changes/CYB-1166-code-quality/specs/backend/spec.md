# Spec — CYB-1166

## Scenario: ValidateCreate rejects unknown asset_type
- **Given** an Asset with asset_type "invalid_type"
- **When** ValidateCreate is called
- **Then** a HierarchyViolation is returned

## Scenario: ParentInfo only contains AssetType
- **Given** an AssetRepoParentGetter query
- **When** GetParentInfo is called
- **Then** only AssetType is populated; LifecycleState, McapFileID, LogicalAssetID, IsCurrent are removed

## Scenario: customerID query failure is logged
- **Given** Customers.Get returns an error
- **When** DeliveryEligibilityProjector.handleEvent processes a tag event
- **Then** slog.Warn is called with the error
