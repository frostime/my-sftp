package client

import (
	"path"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExplicitRemoteFilePreservePath(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{name: "simple relative", source: "a/x.txt", want: "a/x.txt"},
		{name: "dot relative", source: "./a/x.txt", want: "a/x.txt"},
		{name: "parent relative", source: "../a/x.txt", want: preserveParentMarker + "/a/x.txt"},
		{name: "absolute", source: "/etc/ssh/sshd_config", want: "etc/ssh/sshd_config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := explicitRemoteFilePreservePath(tt.source, tt.source)
			if got != tt.want {
				t.Fatalf("explicitRemoteFilePreservePath(%q) = %q, want %q", tt.source, got, tt.want)
			}
		})
	}
}

func TestExplicitRemoteFilePreservePathExpandsHomeSource(t *testing.T) {
	got := explicitRemoteFilePreservePath("~/a.txt", "/home/demo/a.txt")
	if got != "a.txt" {
		t.Fatalf("explicitRemoteFilePreservePath(home) = %q", got)
	}
}

func TestExplicitRemoteFilePreservePathForBareHomeSource(t *testing.T) {
	got := explicitRemoteFilePreservePath("~", "/home/demo")
	if got != "demo" {
		t.Fatalf("explicitRemoteFilePreservePath(bare home) = %q", got)
	}
}

func TestExplicitRemoteFilePreservePathForDirectorySource(t *testing.T) {
	got := explicitRemoteFilePreservePath("a/config", "/srv/a/config")
	if got != "a/config" {
		t.Fatalf("explicitRemoteFilePreservePath(dir) = %q", got)
	}
}

func TestExplicitRemoteFilePreservePathForCurrentDirectorySource(t *testing.T) {
	got := explicitRemoteFilePreservePath(".", "/srv/current")
	if got != "current" {
		t.Fatalf("explicitRemoteFilePreservePath(current dir) = %q", got)
	}
}

func TestExplicitLocalFilePreservePath(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{name: "simple relative", source: filepath.Join("a", "x.txt"), want: "a/x.txt"},
		{name: "parent relative", source: filepath.Join("..", "a", "x.txt"), want: preserveParentMarker + "/a/x.txt"},
		{name: "absolute path", source: filepath.Join(string(filepath.Separator), "work", "a", "x.txt"), want: "work/a/x.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := explicitLocalFilePreservePath(tt.source, tt.source)
			if got != tt.want {
				t.Fatalf("explicitLocalFilePreservePath(%q) = %q, want %q", tt.source, got, tt.want)
			}
		})
	}
}

func TestExplicitLocalFilePreservePathExpandsHomeSource(t *testing.T) {
	resolved := filepath.Join(string(filepath.Separator), "Users", "demo", "a.txt")
	got := explicitLocalFilePreservePath("~/a.txt", resolved)
	if got != "a.txt" {
		t.Fatalf("explicitLocalFilePreservePath(home) = %q", got)
	}
}

func TestExplicitLocalFilePreservePathForBareHomeSource(t *testing.T) {
	resolved := filepath.Join(string(filepath.Separator), "Users", "demo")
	got := explicitLocalFilePreservePath("~", resolved)
	if got != "demo" {
		t.Fatalf("explicitLocalFilePreservePath(bare home) = %q", got)
	}
}

func TestExplicitLocalFilePreservePathForDirectorySource(t *testing.T) {
	resolved := filepath.Join("workspace", "a", "config")
	got := explicitLocalFilePreservePath(filepath.Join("a", "config"), resolved)
	if got != "a/config" {
		t.Fatalf("explicitLocalFilePreservePath(dir) = %q", got)
	}
}

func TestExplicitLocalFilePreservePathForCurrentDirectorySource(t *testing.T) {
	resolved := filepath.Join("workspace", "current")
	got := explicitLocalFilePreservePath(".", resolved)
	if got != "current" {
		t.Fatalf("explicitLocalFilePreservePath(current dir) = %q", got)
	}
}

func TestExplicitLocalFilePreservePathWindowsVolume(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific volume behavior")
	}

	got := explicitLocalFilePreservePath(`C:\work\a\config`, `C:\work\a\config`)
	if got != preserveMetaPrefix+"volume_c__/work/a/config" {
		t.Fatalf("explicitLocalFilePreservePath(volume) = %q", got)
	}
}

func TestExplicitLocalFilePreservePathWindowsVolumeRoot(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific volume behavior")
	}

	got := explicitLocalFilePreservePath(`C:\`, `C:\`)
	if got != preserveMetaPrefix+"volume_c__" {
		t.Fatalf("explicitLocalFilePreservePath(volume root) = %q", got)
	}
}

func TestApplyFlattenMappingDownload(t *testing.T) {
	tasks := []transferTask{
		{remotePath: "/srv/a/readme.md", localPath: filepath.Join("out", "a", "readme.md")},
		{remotePath: "/srv/b/config.yml", localPath: filepath.Join("out", "b", "config.yml")},
	}

	if err := applyFlattenMapping(tasks, "out"); err != nil {
		t.Fatalf("applyFlattenMapping() error = %v", err)
	}

	if tasks[0].localPath != filepath.Join("out", "readme.md") {
		t.Fatalf("first flattened path = %q", tasks[0].localPath)
	}
	if tasks[1].localPath != filepath.Join("out", "config.yml") {
		t.Fatalf("second flattened path = %q", tasks[1].localPath)
	}
}

func TestApplyFlattenMappingUpload(t *testing.T) {
	tasks := []transferTask{
		{localPath: filepath.Join("src", "a", "readme.md"), remotePath: "/dest/src/a/readme.md", isUpload: true},
		{localPath: filepath.Join("src", "b", "config.yml"), remotePath: "/dest/src/b/config.yml", isUpload: true},
	}

	if err := applyFlattenMapping(tasks, "/dest"); err != nil {
		t.Fatalf("applyFlattenMapping() error = %v", err)
	}

	if tasks[0].remotePath != path.Join("/dest", "readme.md") {
		t.Fatalf("first flattened remote path = %q", tasks[0].remotePath)
	}
	if tasks[1].remotePath != path.Join("/dest", "config.yml") {
		t.Fatalf("second flattened remote path = %q", tasks[1].remotePath)
	}
}

func TestApplyFlattenMappingDetectsDuplicateBasename(t *testing.T) {
	tasks := []transferTask{
		{localPath: filepath.Join("src", "a", "readme.md"), remotePath: "/dest/src/a/readme.md", isUpload: true},
		{localPath: filepath.Join("src", "b", "readme.md"), remotePath: "/dest/src/b/readme.md", isUpload: true},
	}

	err := applyFlattenMapping(tasks, "/dest")
	if err == nil {
		t.Fatal("expected duplicate basename error")
	}
	if got, want := err.Error(), "duplicate basename in --flatten mode: readme.md\nHint: remove --flatten or narrow source set"; got != want {
		t.Fatalf("flatten collision error = %q, want %q", got, want)
	}
}

func TestApplyFlattenMappingDetectsWindowsCaseFoldDuplicate(t *testing.T) {
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		t.Skip("case-fold flatten collision behavior is platform-specific")
	}

	tasks := []transferTask{
		{remotePath: "/srv/a/Readme.txt", localPath: filepath.Join("out", "a", "Readme.txt")},
		{remotePath: "/srv/b/README.txt", localPath: filepath.Join("out", "b", "README.txt")},
	}

	err := applyFlattenMapping(tasks, "out")
	if err == nil {
		t.Fatal("expected case-insensitive flatten collision")
	}
}

func TestValidateTargetCollisionsDownload(t *testing.T) {
	tasks := []transferTask{
		{remotePath: "/srv/a/readme.md", localPath: filepath.Join("out", "shared", "readme.md")},
		{remotePath: "/srv/b/readme.md", localPath: filepath.Join("out", "shared", "readme.md")},
	}

	err := validateTargetCollisions(tasks)
	if err == nil {
		t.Fatal("expected duplicate target collision")
	}
}

func TestValidateTargetCollisionsDownloadCaseFold(t *testing.T) {
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		t.Skip("case-fold collision behavior is platform-specific")
	}

	tasks := []transferTask{
		{remotePath: "/srv/a/readme.md", localPath: filepath.Join("out", "shared", "Readme.txt")},
		{remotePath: "/srv/b/readme.md", localPath: filepath.Join("out", "shared", "README.txt")},
	}

	err := validateTargetCollisions(tasks)
	if err == nil {
		t.Fatal("expected case-insensitive duplicate target collision")
	}
}

func TestValidateTargetCollisionsDownloadPrefixConflict(t *testing.T) {
	tasks := []transferTask{
		{remotePath: "/srv/a", localPath: filepath.Join("out", "shared", "a")},
		{remotePath: "/srv/a/b.txt", localPath: filepath.Join("out", "shared", "a", "b.txt")},
	}

	err := validateTargetCollisions(tasks)
	if err == nil {
		t.Fatal("expected prefix target collision")
	}
}

func TestValidateTargetCollisionsUpload(t *testing.T) {
	tasks := []transferTask{
		{localPath: filepath.Join("src", "a", "x.txt"), remotePath: "/dest/shared/x.txt", isUpload: true},
		{localPath: filepath.Join("src", "b", "x.txt"), remotePath: "/dest/shared/x.txt", isUpload: true},
	}

	err := validateTargetCollisions(tasks)
	if err == nil {
		t.Fatal("expected duplicate target collision")
	}
}

func TestValidateTargetCollisionsUploadPrefixConflict(t *testing.T) {
	tasks := []transferTask{
		{localPath: filepath.Join("src", "a"), remotePath: "/dest/shared/a", isUpload: true},
		{localPath: filepath.Join("src", "b.txt"), remotePath: "/dest/shared/a/b.txt", isUpload: true},
	}

	err := validateTargetCollisions(tasks)
	if err == nil {
		t.Fatal("expected prefix target collision")
	}
}

func TestExplicitPreservePathKeepsDistinctParentPrefixes(t *testing.T) {
	remoteA := explicitRemoteFilePreservePath("../cfg", "/srv/cfg")
	remoteB := explicitRemoteFilePreservePath("../../cfg", "/srv/cfg")
	if remoteA == remoteB {
		t.Fatalf("remote preserve paths collapsed: %q", remoteA)
	}

	localA := explicitLocalFilePreservePath(filepath.Join("..", "cfg"), filepath.Join("workspace", "cfg"))
	localB := explicitLocalFilePreservePath(filepath.Join("..", "..", "cfg"), filepath.Join("workspace", "cfg"))
	if localA == localB {
		t.Fatalf("local preserve paths collapsed: %q", localA)
	}
}

func TestUsesReservedPreservePrefix(t *testing.T) {
	if !usesReservedPreservePrefix(preserveParentMarker+"/cfg", false) {
		t.Fatal("expected reserved prefix detection for remote source")
	}
	if !usesReservedPreservePrefix(preserveParentMarker+"/*", false) {
		t.Fatal("expected reserved prefix detection for remote glob source")
	}
	if !usesReservedPreservePrefix("../"+preserveParentMarker+"/cfg", false) {
		t.Fatal("expected reserved prefix detection for nested parent marker")
	}
	if !usesReservedPreservePrefix(preserveMetaPrefix+"volume_c__/cfg", true) {
		t.Fatal("expected reserved prefix detection for local source")
	}
	if runtime.GOOS == "windows" && usesReservedPreservePrefix(`C:\__my_sftp_parent__\*`, true) {
		t.Fatal("did not expect absolute Windows volume source to be rejected")
	}
	if usesReservedPreservePrefix("../cfg", false) {
		t.Fatal("did not expect parent-relative source to count as reserved prefix")
	}
}
