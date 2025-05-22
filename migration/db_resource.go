// Package migration 提供了一个灵活的版本控制系统，用于管理数据库和配置文件的升级。
package migration

import (
	"gorm.io/gorm"
)

// DBResource 表示数据库升级资源，用于执行数据库相关的版本升级和回滚操作
type DBResource struct {
	db    *gorm.DB // 数据库连接
	query string   // 要执行的SQL查询
}

// NewDBResource 创建一个新的数据库升级资源
// db: 数据库连接
// query: 要执行的SQL查询语句
func NewDBResource(db *gorm.DB, query string) *DBResource {
	return &DBResource{
		db:    db,
		query: query,
	}
}

// Apply 执行数据库升级操作
// 直接执行SQL查询，不进行特殊处理
func (r *DBResource) Apply() error {
	return r.db.Exec(r.query).Error
}

// Rollback 数据库操作的回滚（如果需要）
// 数据库操作通常不需要特殊的回滚处理，因为回滚操作会通过 Down 资源执行
func (r *DBResource) Rollback() error {
	// 数据库操作通常不需要特殊的回滚处理
	// 因为回滚操作会通过 Down 资源执行
	return nil
}
