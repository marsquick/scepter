package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"scepter/migration"
)

// FileResource 实现了 migration.Resource 接口
// 用于演示如何创建自定义资源类型
type FileResource struct {
	Path    string
	Content string
}

// Apply 实现 Resource 接口的 Apply 方法
// 用于执行版本升级操作
func (r *FileResource) Apply() error {
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(r.Path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	return os.WriteFile(r.Path, []byte(r.Content), 0644)
}

// Rollback 实现 Resource 接口的 Rollback 方法
// 用于执行版本回滚操作
func (r *FileResource) Rollback() error {
	return os.Remove(r.Path)
}

func main() {
	// 创建全局管理器，指定版本记录文件路径
	gm := migration.NewGlobalManager("migrations.yaml")

	// 注册数据库模块
	dbManager := gm.RegisterModule("database")

	// 添加数据库模块的版本定义
	dbManager.AddVersion(migration.VersionDefinition{
		Version: "1.0.0",
		Up: &FileResource{
			Path:    "db/schema.sql",
			Content: "CREATE TABLE users (id INT, name VARCHAR(255));",
		},
		Down: &FileResource{
			Path:    "db/schema.sql",
			Content: "DROP TABLE users;",
		},
		Timeout: 5 * time.Second,
	})

	// 注册配置模块
	configManager := gm.RegisterModule("config")

	// 添加配置模块的版本定义
	configManager.AddVersion(migration.VersionDefinition{
		Version: "1.0.0",
		Up: &FileResource{
			Path:    "config/app.yaml",
			Content: "app:\n  name: MyApp\n  version: 1.0.0",
		},
		Down: &FileResource{
			Path:    "config/app.yaml",
			Content: "",
		},
		Timeout: 5 * time.Second,
	})

	// 设置模块执行顺序：数据库模块在配置模块之前执行
	err := gm.SetModuleOrder("database", migration.OrderDefinition{
		Type:    migration.Before,
		Modules: []string{"config"},
	})
	if err != nil {
		log.Fatalf("Failed to set module order: %v", err)
	}

	// 执行所有版本升级
	if err := gm.RunAllVersions(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	fmt.Println("All migrations completed successfully!")
}
