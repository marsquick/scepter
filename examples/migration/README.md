# Migration 示例

这个示例展示了如何使用 scepter 的 migration 包来管理数据库和配置文件的版本控制。

## 功能特点

- 多模块版本控制
- 版本依赖管理
- 执行顺序控制
- 自动回滚
- 版本完整性检查

## 示例说明

这个示例包含两个模块：

1. **数据库模块**：
   - 创建用户表
   - 提供回滚操作（删除表）

2. **配置模块**：
   - 创建应用配置文件
   - 提供回滚操作（清空配置）

## 运行示例

```bash
# 在项目根目录下运行
go run examples/migration/main.go
```

## 预期输出

运行成功后，将创建以下文件：

1. `db/schema.sql`：包含创建用户表的 SQL 语句
2. `config/app.yaml`：包含应用配置
3. `migrations.yaml`：记录版本执行历史

## 自定义资源

示例中的 `FileResource` 展示了如何实现自定义资源类型：

```go
type FileResource struct {
    Path    string
    Content string
}

func (r *FileResource) Apply() error {
    // 实现升级操作
}

func (r *FileResource) Rollback() error {
    // 实现回滚操作
}
```

## 注意事项

1. 确保有适当的文件系统权限
2. 版本号应该遵循语义化版本规范
3. 建议为每个版本设置合理的超时时间
4. 在生产环境中应该实现更复杂的资源类型 