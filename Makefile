.PHONY: build clean run help windows linux darwin

# 默认目标
all: build

# 编译当前平台
build:
	go build -o my-sftp

# Windows 64位
windows:
	GOOS=windows GOARCH=amd64 go build -o my-sftp.exe

# Linux 64位
linux:
	GOOS=linux GOARCH=amd64 go build -o my-sftp-linux

# macOS 64位
darwin:
	GOOS=darwin GOARCH=amd64 go build -o my-sftp-darwin

# 交叉编译所有平台
build-all: windows linux darwin
	@echo "All platforms built successfully"

# 清理编译产物
clean:
	rm -f my-sftp my-sftp.exe my-sftp-linux my-sftp-darwin

# 运行（需先配置参数）
run:
	go run main.go -host $(HOST) -user $(USER)

# 安装依赖
deps:
	go mod download
	go mod tidy

# 帮助信息
help:
	@echo "Available targets:"
	@echo "  make build       - Build for current platform"
	@echo "  make windows     - Build for Windows (amd64)"
	@echo "  make linux       - Build for Linux (amd64)"
	@echo "  make darwin      - Build for macOS (amd64)"
	@echo "  make build-all   - Build for all platforms"
	@echo "  make clean       - Remove build artifacts"
	@echo "  make deps        - Download dependencies"
	@echo "  make run         - Run with HOST=xxx USER=xxx"
