# Tasks — CYB-3297 Phase B (env facet)

- [x] query_ir.go: facetFieldPath 加 env→metadata.env
- [x] query_field_registry.yaml: 登记 env（filter+facet on ES）
- [x] reducer: ASSET_DISCOVERY_FACETS 加 env；mapFacetFieldToAggregationKey env→env_agg
- [x] AssetsFacetSidebar: AGG_KEY_MAP env_agg→env；env 选项动态化（不再用 ENV_OPTIONS）
- [x] go build + es/queryplan/query 测试通过；前端 build + biome 通过
- [ ] 合 dev → 部署 → dev 验证：环境 facet 显示真实值（工作区/家庭…）带计数；点选按 env 过滤
