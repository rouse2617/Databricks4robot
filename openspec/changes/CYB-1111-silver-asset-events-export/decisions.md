## 2026-05-23 — OpenSpec approval and compatible API shape
- **Context**: CYB-1111 extends `/lakehouse/tables` visibility for Silver asset-events without changing the response shape.
- **Decision**: Proceed after user confirmed "可以,开始弄吧"; keep SDK and Frontend out of scope because `items[].table_name,row_count` remains compatible.
- **Alternatives**: Add a new endpoint or typed SDK helper for Silver table state.
- **Rationale**: Existing clients can already consume additional table rows; documenting the optional row and smoke shape check is sufficient for this issue.
