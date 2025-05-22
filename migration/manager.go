// Package migration 提供了一个灵活的版本控制系统，用于管理数据库和配置文件的升级。
// 它支持：
// - 多模块版本控制
// - 版本依赖管理
// - 执行顺序控制
// - 自动回滚
// - 版本完整性检查
// - 前置条件验证
package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// VersionStatus 表示版本执行的状态
type VersionStatus int

const (
	Pending    VersionStatus = iota // 等待执行
	Running                         // 正在执行
	Completed                       // 执行完成
	Failed                          // 执行失败
	RolledBack                      // 已回滚
)

// VersionDefinition 定义了一个版本的所有信息
type VersionDefinition struct {
	Version      string        // 版本号
	Up           Resource      // 升级操作
	Down         Resource      // 回滚操作
	Checksum     string        // 版本内容的校验和
	Validate     func() error  // 版本验证函数
	PreCheck     func() error  // 前置条件检查函数
	Dependencies []string      // 依赖的其他版本
	Timeout      time.Duration // 版本执行超时时间
}

// FileVersionRecord 记录已执行版本的信息
type FileVersionRecord struct {
	Version   string        `yaml:"version"`         // 版本号
	AppliedAt time.Time     `yaml:"applied_at"`      // 执行时间
	Status    VersionStatus `yaml:"status"`          // 执行状态
	Error     string        `yaml:"error,omitempty"` // 错误信息（如果有）
}

// ModuleRecord 记录模块的版本执行历史
type ModuleRecord struct {
	Versions []FileVersionRecord `yaml:"versions"` // 版本执行记录列表
}

// VersionRecords 记录所有模块的版本执行历史
type VersionRecords struct {
	Modules map[string]ModuleRecord `yaml:"modules"` // 模块名称到记录的映射
}

// Manager 管理单个模块的版本控制
type Manager struct {
	module     string              // 模块名称
	versions   []VersionDefinition // 版本定义列表
	recordPath string              // 版本记录文件路径
}

// NewManager 创建一个新的版本控制管理器
func NewManager(module, recordPath string) *Manager {
	return &Manager{
		module:     module,
		versions:   make([]VersionDefinition, 0),
		recordPath: recordPath,
	}
}

// AddVersion 添加一个新的版本定义
func (m *Manager) AddVersion(version VersionDefinition) {
	m.versions = append(m.versions, version)
}

// loadVersionRecords 从文件加载版本记录
func (m *Manager) loadVersionRecords() (*VersionRecords, error) {
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(m.recordPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create version records directory: %v", err)
	}

	// 如果文件不存在，返回空记录
	if _, err := os.Stat(m.recordPath); os.IsNotExist(err) {
		return &VersionRecords{
			Modules: make(map[string]ModuleRecord),
		}, nil
	}

	// 读取文件
	data, err := os.ReadFile(m.recordPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read version records: %v", err)
	}

	var records VersionRecords
	if err := yaml.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("failed to parse version records: %v", err)
	}

	// 确保模块记录存在
	if records.Modules == nil {
		records.Modules = make(map[string]ModuleRecord)
	}

	return &records, nil
}

// saveVersionRecords 保存版本记录到文件
func (m *Manager) saveVersionRecords(records *VersionRecords) error {
	data, err := yaml.Marshal(records)
	if err != nil {
		return fmt.Errorf("failed to marshal version records: %v", err)
	}

	if err := os.WriteFile(m.recordPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write version records: %v", err)
	}

	return nil
}

// RunVersions 执行所有未执行的版本
func (m *Manager) RunVersions() error {
	// 加载版本记录
	records, err := m.loadVersionRecords()
	if err != nil {
		return fmt.Errorf("failed to load version records: %v", err)
	}

	// 获取当前模块的记录
	moduleRecord, exists := records.Modules[m.module]
	if !exists {
		moduleRecord = ModuleRecord{
			Versions: make([]FileVersionRecord, 0),
		}
	}

	// 创建已执行版本的map，用于快速查找
	executedMap := make(map[string]bool)
	for _, record := range moduleRecord.Versions {
		executedMap[record.Version] = true
	}

	// 按顺序执行未执行的版本
	for _, version := range m.versions {
		if !executedMap[version.Version] {
			log.Printf("Executing version: %s for module: %s", version.Version, m.module)

			// 执行版本升级
			if err := version.Up.Apply(); err != nil {
				return fmt.Errorf("failed to execute version %s: %v", version.Version, err)
			}

			// 记录版本
			record := FileVersionRecord{
				Version:   version.Version,
				AppliedAt: time.Now(),
			}
			moduleRecord.Versions = append(moduleRecord.Versions, record)
			records.Modules[m.module] = moduleRecord

			// 保存版本记录
			if err := m.saveVersionRecords(records); err != nil {
				// 如果保存记录失败，尝试回滚
				if rollbackErr := version.Up.Rollback(); rollbackErr != nil {
					log.Printf("Warning: failed to rollback version %s: %v", version.Version, rollbackErr)
				}
				return fmt.Errorf("failed to save version record: %v", err)
			}

			log.Printf("Successfully executed version: %s for module: %s", version.Version, m.module)
		}
	}

	return nil
}

// RollbackVersion 回滚最后一个版本
func (m *Manager) RollbackVersion() error {
	// 加载版本记录
	records, err := m.loadVersionRecords()
	if err != nil {
		return fmt.Errorf("failed to load version records: %v", err)
	}

	// 获取当前模块的记录
	moduleRecord, exists := records.Modules[m.module]
	if !exists || len(moduleRecord.Versions) == 0 {
		return fmt.Errorf("no versions to rollback for module: %s", m.module)
	}

	// 获取最后一个版本记录
	lastRecord := moduleRecord.Versions[len(moduleRecord.Versions)-1]

	// 找到对应的版本
	var targetVersion *VersionDefinition
	for _, v := range m.versions {
		if v.Version == lastRecord.Version {
			targetVersion = &v
			break
		}
	}

	if targetVersion == nil {
		return fmt.Errorf("version %s not found for module: %s", lastRecord.Version, m.module)
	}

	// 执行回滚
	if err := targetVersion.Down.Apply(); err != nil {
		return fmt.Errorf("failed to rollback version %s: %v", lastRecord.Version, err)
	}

	// 更新版本记录
	moduleRecord.Versions = moduleRecord.Versions[:len(moduleRecord.Versions)-1]
	records.Modules[m.module] = moduleRecord
	if err := m.saveVersionRecords(records); err != nil {
		return fmt.Errorf("failed to update version records: %v", err)
	}

	log.Printf("Successfully rolled back version: %s for module: %s", lastRecord.Version, m.module)
	return nil
}

// OrderType 定义模块执行顺序的类型
type OrderType int

const (
	After  OrderType = iota // 在指定模块之后执行
	Before                  // 在指定模块之前执行
)

// OrderDefinition 定义模块执行顺序
type OrderDefinition struct {
	Type    OrderType // 顺序类型
	Modules []string  // 相关模块列表
}

// GlobalManager 管理所有模块的版本控制
type GlobalManager struct {
	managers     map[string]*Manager // 模块名称到管理器的映射
	recordPath   string              // 版本记录文件路径
	mu           sync.Mutex          // 并发锁
	moduleOrders []ModuleOrder       // 模块执行顺序
	lockFile     string              // 锁文件路径
}

// NewGlobalManager 创建一个新的全局版本控制管理器
func NewGlobalManager(recordPath string) *GlobalManager {
	return &GlobalManager{
		managers:     make(map[string]*Manager),
		recordPath:   recordPath,
		moduleOrders: make([]ModuleOrder, 0),
		lockFile:     recordPath + ".lock",
	}
}

// RegisterModule 注册一个新的模块
func (gm *GlobalManager) RegisterModule(module string) *Manager {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if manager, exists := gm.managers[module]; exists {
		return manager
	}

	manager := NewManager(module, gm.recordPath)
	gm.managers[module] = manager
	return manager
}

// SetModuleOrder 设置模块的执行顺序
func (gm *GlobalManager) SetModuleOrder(moduleToExecute string, order OrderDefinition) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// 验证要执行的模块是否已注册
	if _, exists := gm.managers[moduleToExecute]; !exists {
		return fmt.Errorf("module to execute %s not registered", moduleToExecute)
	}

	// 验证所有依赖模块是否都已注册
	for _, m := range order.Modules {
		if _, exists := gm.managers[m]; !exists {
			return fmt.Errorf("dependency module %s not registered", m)
		}
	}

	// 检查循环依赖
	if err := gm.checkCircularDependency(moduleToExecute, order); err != nil {
		return err
	}

	var before, after []string
	if order.Type == Before {
		before = order.Modules
	} else {
		after = order.Modules
	}

	gm.moduleOrders = append(gm.moduleOrders, ModuleOrder{
		Module: moduleToExecute,
		Before: before,
		After:  after,
	})
	return nil
}

// checkCircularDependency 检查是否存在循环依赖
func (gm *GlobalManager) checkCircularDependency(module string, order OrderDefinition) error {
	visited := make(map[string]bool)
	path := make(map[string]bool)

	var check func(string) error
	check = func(m string) error {
		if path[m] {
			return fmt.Errorf("circular dependency detected: %s", m)
		}
		if visited[m] {
			return nil
		}

		visited[m] = true
		path[m] = true

		// 检查所有依赖
		for _, order := range gm.moduleOrders {
			if order.Module == m {
				// 检查 before 依赖
				for _, b := range order.Before {
					if err := check(b); err != nil {
						return err
					}
				}
				// 检查 after 依赖
				for _, a := range order.After {
					if err := check(a); err != nil {
						return err
					}
				}
			}
		}

		path[m] = false
		return nil
	}

	return check(module)
}

// calculateChecksum 计算版本的校验和
func calculateChecksum(version VersionDefinition) string {
	// 将版本内容序列化为YAML
	data, err := yaml.Marshal(version)
	if err != nil {
		return ""
	}

	// 计算SHA256校验和
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// acquireLock 获取版本执行锁
func (gm *GlobalManager) acquireLock() error {
	// 检查是否已有其他进程在运行版本升级
	if _, err := os.Stat(gm.lockFile); err == nil {
		return fmt.Errorf("another migration process is running")
	}

	// 创建锁文件
	return os.WriteFile(gm.lockFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
}

// releaseLock 释放版本执行锁
func (gm *GlobalManager) releaseLock() error {
	return os.Remove(gm.lockFile)
}

// checkVersionDependencies 检查版本依赖
func (gm *GlobalManager) checkVersionDependencies(module string, version VersionDefinition, records *VersionRecords) error {
	moduleRecord := records.Modules[module]
	executedVersions := make(map[string]bool)
	for _, v := range moduleRecord.Versions {
		executedVersions[v.Version] = true
	}

	// 检查所有依赖版本是否已执行
	for _, dep := range version.Dependencies {
		if !executedVersions[dep] {
			return fmt.Errorf("version %s depends on version %s which has not been executed",
				version.Version, dep)
		}
	}
	return nil
}

// checkVersionStatus 检查版本执行状态
func (gm *GlobalManager) checkVersionStatus(module string, version string, records *VersionRecords) error {
	moduleRecord := records.Modules[module]
	for _, v := range moduleRecord.Versions {
		if v.Version == version {
			if v.Status == Failed {
				return fmt.Errorf("version %s previously failed: %s", version, v.Error)
			}
			if v.Status == Running {
				return fmt.Errorf("version %s is currently running", version)
			}
		}
	}
	return nil
}

// RunAllVersions 执行所有模块的版本升级
func (gm *GlobalManager) RunAllVersions() error {
	// 获取执行锁
	if err := gm.acquireLock(); err != nil {
		return fmt.Errorf("failed to acquire lock: %v", err)
	}
	defer gm.releaseLock()

	// 加载版本记录
	records, err := gm.loadVersionRecords()
	if err != nil {
		return fmt.Errorf("failed to load version records: %v", err)
	}

	// 构建依赖图
	graph := make(map[string][]string)
	for _, order := range gm.moduleOrders {
		// 添加 before 依赖
		for _, before := range order.Before {
			graph[order.Module] = append(graph[order.Module], before)
		}
		// 添加 after 依赖
		for _, after := range order.After {
			graph[after] = append(graph[after], order.Module)
		}
	}

	// 获取执行顺序
	order, err := gm.topologicalSort(graph)
	if err != nil {
		return fmt.Errorf("failed to determine execution order: %v", err)
	}

	// 按顺序执行模块
	for _, module := range order {
		manager := gm.managers[module]
		log.Printf("Processing module: %s", module)

		// 获取模块记录
		moduleRecord, exists := records.Modules[module]
		if !exists {
			moduleRecord = ModuleRecord{
				Versions: make([]FileVersionRecord, 0),
			}
		}

		// 创建已执行版本的map
		executedMap := make(map[string]bool)
		for _, record := range moduleRecord.Versions {
			executedMap[record.Version] = true
		}

		// 执行未执行的版本
		for _, version := range manager.versions {
			if !executedMap[version.Version] {
				// 检查版本完整性
				if version.Validate != nil {
					if err := version.Validate(); err != nil {
						return fmt.Errorf("version %s validation failed: %v", version.Version, err)
					}
				}

				// 执行前置条件检查
				if version.PreCheck != nil {
					if err := version.PreCheck(); err != nil {
						return fmt.Errorf("version %s pre-check failed: %v", version.Version, err)
					}
				}

				// 验证校验和
				if version.Checksum != "" {
					currentChecksum := calculateChecksum(version)
					if currentChecksum != version.Checksum {
						return fmt.Errorf("version %s checksum mismatch", version.Version)
					}
				}

				// 检查版本依赖
				if err := gm.checkVersionDependencies(module, version, records); err != nil {
					return err
				}

				// 检查版本状态
				if err := gm.checkVersionStatus(module, version.Version, records); err != nil {
					return err
				}

				log.Printf("Executing version: %s for module: %s", version.Version, module)

				// 更新版本状态为运行中
				record := FileVersionRecord{
					Version:   version.Version,
					AppliedAt: time.Now(),
					Status:    Running,
				}
				moduleRecord.Versions = append(moduleRecord.Versions, record)
				records.Modules[module] = moduleRecord
				if err := gm.saveVersionRecords(records); err != nil {
					return fmt.Errorf("failed to save version record: %v", err)
				}

				// 使用超时控制执行版本升级
				ctx, cancel := context.WithTimeout(context.Background(), version.Timeout)
				defer cancel()

				done := make(chan error, 1)
				go func() {
					done <- version.Up.Apply()
				}()

				var execErr error
				select {
				case err := <-done:
					execErr = err
				case <-ctx.Done():
					execErr = fmt.Errorf("version %s execution timed out after %v",
						version.Version, version.Timeout)
				}

				// 更新版本状态
				if execErr != nil {
					record.Status = Failed
					record.Error = execErr.Error()
				} else {
					record.Status = Completed
				}
				records.Modules[module] = moduleRecord
				if err := gm.saveVersionRecords(records); err != nil {
					return fmt.Errorf("failed to save version record: %v", err)
				}

				if execErr != nil {
					return fmt.Errorf("failed to execute version %s for module %s: %v",
						version.Version, module, execErr)
				}

				log.Printf("Successfully executed version: %s for module: %s",
					version.Version, module)
			}
		}
	}

	return nil
}

// topologicalSort 使用拓扑排序确定模块执行顺序
func (gm *GlobalManager) topologicalSort(graph map[string][]string) ([]string, error) {
	visited := make(map[string]bool)
	temp := make(map[string]bool)
	order := make([]string, 0)

	var visit func(string) error
	visit = func(node string) error {
		if temp[node] {
			return fmt.Errorf("circular dependency detected")
		}
		if visited[node] {
			return nil
		}

		temp[node] = true

		for _, neighbor := range graph[node] {
			if err := visit(neighbor); err != nil {
				return err
			}
		}

		temp[node] = false
		visited[node] = true
		order = append([]string{node}, order...)
		return nil
	}

	// 访问所有模块
	for module := range gm.managers {
		if !visited[module] {
			if err := visit(module); err != nil {
				return nil, err
			}
		}
	}

	return order, nil
}

// loadVersionRecords 从文件加载版本记录
func (gm *GlobalManager) loadVersionRecords() (*VersionRecords, error) {
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(gm.recordPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create version records directory: %v", err)
	}

	// 如果文件不存在，返回空记录
	if _, err := os.Stat(gm.recordPath); os.IsNotExist(err) {
		return &VersionRecords{
			Modules: make(map[string]ModuleRecord),
		}, nil
	}

	// 读取文件
	data, err := os.ReadFile(gm.recordPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read version records: %v", err)
	}

	var records VersionRecords
	if err := yaml.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("failed to parse version records: %v", err)
	}

	// 确保模块记录存在
	if records.Modules == nil {
		records.Modules = make(map[string]ModuleRecord)
	}

	return &records, nil
}

// saveVersionRecords 保存版本记录到文件
func (gm *GlobalManager) saveVersionRecords(records *VersionRecords) error {
	data, err := yaml.Marshal(records)
	if err != nil {
		return fmt.Errorf("failed to marshal version records: %v", err)
	}

	if err := os.WriteFile(gm.recordPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write version records: %v", err)
	}

	return nil
}
