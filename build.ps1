$Version = git describe --tags --always
$Commit = git rev-parse --short HEAD
$Date = Get-Date -Format "yyyy-MM-dd"

go build -ldflags "-X 'main.Version=$Version' -X 'main.Commit=$Commit' -X 'main.Date=$Date'" -o my-sftp.exe
