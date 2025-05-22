// Package migration 提供了一个灵活的版本控制系统，用于管理数据库和配置文件的升级。
package migration

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileResource 表示文件升级资源，用于执行文件相关的版本升级和回滚操作
type FileResource struct {
	path    string // 文件路径
	content string // 文件内容
}

// NewFileResource 创建一个新的文件升级资源
// path: 文件路径
// content: 文件内容
func NewFileResource(path, content string) *FileResource {
	return &FileResource{
		path:    path,
		content: content,
	}
}

// Apply 执行文件升级操作
// 1. 确保目标目录存在
// 2. 如果文件已存在，创建备份
// 3. 写入新文件内容
func (r *FileResource) Apply() error {
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(r.path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// 如果文件已存在，创建备份
	if _, err := os.Stat(r.path); err == nil {
		backupPath := r.path + ".bak"
		if err := os.Rename(r.path, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %v", err)
		}
	}

	// 写入新文件
	return os.WriteFile(r.path, []byte(r.content), 0644)
}

// Rollback 执行文件回滚操作
// 1. 检查备份文件是否存在
// 2. 恢复备份文件到原始位置
func (r *FileResource) Rollback() error {
	backupPath := r.path + ".bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	// 恢复备份文件
	return os.Rename(backupPath, r.path)
}
