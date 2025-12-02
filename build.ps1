$ErrorActionPreference = "Stop"  # 遇到错误立即停止

try {
    $Version = git describe --tags --always
    $Commit = git rev-parse --short HEAD
    $Date = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

    Write-Host "Building my-sftp version $Version (commit $Commit) on $Date" -ForegroundColor Green

    go build -ldflags "-s -w -X 'main.Version=$Version' -X 'main.Commit=$Commit' -X 'main.Date=$Date'" -o my-sftp.exe

    if ($LASTEXITCODE -ne 0) {
        throw "Build failed with exit code $LASTEXITCODE"
    }

    Write-Host "Build successful: my-sftp.exe" -ForegroundColor Green
}
catch {
    Write-Error "Build failed: $_"
    exit 1
}
