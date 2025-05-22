#!/bin/bash

# 设置错误时退出
set -e

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 打印带颜色的信息
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 获取脚本所在目录的绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# 切换到项目根目录
cd "$PROJECT_ROOT"

# 检查是否存在 examples 目录
if [ ! -d "examples" ]; then
    error "examples directory not found"
    exit 1
fi

# 创建构建输出目录
BUILD_DIR="build/examples"
mkdir -p "$BUILD_DIR"

# 编译所有示例
info "Building all examples..."

# 遍历 examples 目录下的所有子目录
for example_dir in examples/*/; do
    if [ -f "${example_dir}main.go" ]; then
        example_name=$(basename "$example_dir")
        info "Building example: $example_name"
        
        # 编译示例
        if go build -o "$BUILD_DIR/$example_name" "$example_dir"main.go; then
            info "Successfully built $example_name"
        else
            error "Failed to build $example_name"
            exit 1
        fi
    fi
done

info "All examples built successfully!"
info "Build artifacts are in: $BUILD_DIR"

# 列出所有构建的二进制文件
echo -e "\nBuilt binaries:"
ls -l "$BUILD_DIR" 