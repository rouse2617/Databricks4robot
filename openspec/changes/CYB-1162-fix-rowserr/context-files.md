# Context files — CYB-1162

backend/internal/handlers/asset/lineage_response.go  # 目标文件：三处 rows.Next() 循环
backend/internal/handlers/asset/handler_test.go       # 已有 fakeAssetSQLRows / fakeAssetSQLQuerier 测试辅助工具
backend/internal/handlers/audit/handler.go             # 参考实现：commit 7f0cf5e 已修复的 rows.Err 检查模式
