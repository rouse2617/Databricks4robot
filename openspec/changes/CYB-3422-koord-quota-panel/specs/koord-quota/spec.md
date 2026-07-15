# Koordinator ElasticQuota 只读展示 · Spec Delta

## ADDED Requirements

### Requirement: 用户可以在 Registry 页面查看 ElasticQuota 资源池状态

用户访问 `/registry` → pools tab 时,能看到集群里配置的所有 ElasticQuota 资源池的实时状态(名字 / namespace / CPU min-max-used / memory min-max-used / usage %)。

#### Scenario: 集群装了 Koordinator + ElasticQuota

- **GIVEN** 集群里有 3 个 ElasticQuota(`cyberorigin-delivery-high` / `-mid` / `-low`)
- **WHEN** 用户访问 `/registry` → pools tab
- **THEN** 页面显示「ElasticQuota 资源池」section,表格有 3 行,每行显示 name / namespace / CPU(min=4/max=24/used=X + usage bar) / memory / 最近刷新时间
- **AND** 面板每 15 秒自动刷新一次

#### Scenario: 集群未装 Koordinator(ElasticQuota CRD 不存在)

- **GIVEN** 集群里没有 `elasticquotas.scheduling.sigs.k8s.io` CRD
- **WHEN** 用户访问 `/registry` → pools tab
- **THEN** ElasticQuota section 完全不显示(不显示空表格,不显示错误)
- **AND** ExecutionTarget 表格照常显示(不受影响)

#### Scenario: K8s API 短暂不可达

- **GIVEN** K8s API server 短暂 503
- **WHEN** 前端请求 `GET /api/v1/elastic-quotas`
- **THEN** 后端返回 502,前端 section 显示「加载失败」加重试按钮,不 crash

### Requirement: 后端提供 `GET /api/v1/elastic-quotas` API

后端提供只读接口返回 ElasticQuota 列表。

#### Scenario: 正常返回

- **WHEN** curl `GET /api/v1/elastic-quotas`
- **THEN** 返回 200 + JSON:
  ```json
  {
    "items": [
      {
        "name": "cyberorigin-delivery-high",
        "namespace": "cyber-databrew-dev",
        "min": {"cpu": "4", "memory": "8Gi"},
        "max": {"cpu": "24", "memory": "48Gi"},
        "used": {"cpu": "0", "memory": "0"},
        "utilizationPercent": {"cpu": 0, "memory": 0}
      }
    ]
  }
  ```

#### Scenario: CRD 不存在

- **GIVEN** 集群里 `elasticquotas.scheduling.sigs.k8s.io` CRD 不存在
- **WHEN** curl `GET /api/v1/elastic-quotas`
- **THEN** 返回 200 + `{"items": []}`

## RELATED

- 前置:CYB-3409(Reservation PoC,已 Done);CYB-3420(ElasticQuota 生产部署,已 Done)
- 后续:CYB-3422 P3.1b(schema + transpiler 注入,本次不做)
