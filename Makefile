# .PHONY: build clean run help windows linux darwin

# # 默认目标
# all: windows

# # Windows 64位
# windows:
# 	GOOS=windows GOARCH=amd64 go build -o my-sftp.exe

# # Linux 64位
# linux:
# 	GOOS=linux GOARCH=amd64 go build -o my-sftp-linux

# # macOS 64位
# darwin:
# 	GOOS=darwin GOARCH=amd64 go build -o my-sftp-darwin

# # 交叉编译所有平台
# build-all: windows linux darwin
# 	@echo "All platforms built successfully"

# # 清理编译产物
# clean:
# 	rm -f my-sftp my-sftp.exe my-sftp-linux my-sftp-darwin
