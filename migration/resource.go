// Package migration 提供了一个灵活的版本控制系统，用于管理数据库和配置文件的升级。
package migration

// Resource 定义了版本升级和回滚操作的接口
type Resource interface {
	Apply() error    // 执行升级或回滚操作
	Rollback() error // 回滚操作（如果需要）
}

// ModuleOrder 定义模块的执行顺序
type ModuleOrder struct {
	Module string   // 当前模块
	Before []string // 在当前模块之前执行的模块
	After  []string // 在当前模块之后执行的模块
}
