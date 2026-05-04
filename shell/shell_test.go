package shell

import (
	"testing"

	"github.com/frostime/my-sftp/client"
)

func TestParseTransferCLIArgsSupportsDashLeadingSourceWithTerminator(t *testing.T) {
	opts, err := parseTransferCLIArgs([]string{"-d", "out", "--", "-report.txt"})
	if err != nil {
		t.Fatalf("parseTransferCLIArgs() error = %v", err)
	}
	if opts.targetDir != "out" {
		t.Fatalf("targetDir = %q, want out", opts.targetDir)
	}
	if len(opts.sources) != 1 || opts.sources[0] != "-report.txt" {
		t.Fatalf("sources = %#v", opts.sources)
	}
}

func TestParseTransferCLIArgsRejectsDashLeadingSourceWithoutTerminator(t *testing.T) {
	if _, err := parseTransferCLIArgs([]string{"-report.txt"}); err == nil {
		t.Fatal("expected unknown option error")
	}
}

func TestParseTransferCLIArgsKeepsOptionOrderFlexible(t *testing.T) {
	opts, err := parseTransferCLIArgs([]string{"src.txt", "--flatten", "-d", "out"})
	if err != nil {
		t.Fatalf("parseTransferCLIArgs() error = %v", err)
	}
	if !opts.flatten {
		t.Fatal("expected flatten option to be set")
	}
	if opts.targetDir != "out" {
		t.Fatalf("targetDir = %q, want out", opts.targetDir)
	}
	if len(opts.sources) != 1 || opts.sources[0] != "src.txt" {
		t.Fatalf("sources = %#v", opts.sources)
	}
}

func TestValidateTransferRename(t *testing.T) {
	tests := []struct {
		name    string
		rename  string
		wantErr bool
	}{
		{name: "plain filename", rename: "report.csv"},
		{name: "dot", rename: ".", wantErr: true},
		{name: "dotdot", rename: "..", wantErr: true},
		{name: "slash path", rename: "nested/report.csv", wantErr: true},
		{name: "backslash path", rename: `nested\report.csv`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTransferRename(tt.rename)
			if tt.wantErr && err == nil {
				t.Fatalf("validateTransferRename(%q) expected error", tt.rename)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validateTransferRename(%q) error = %v", tt.rename, err)
			}
		})
	}
}

func TestBuildDownloadCommandOptions(t *testing.T) {
	parsed := &transferCLIOptions{recursive: true, flatten: true}
	got := buildDownloadCommandOptions(parsed)
	want := &client.DownloadOptions{
		Recursive:    true,
		ShowProgress: true,
		Concurrency:  client.MaxConcurrentTransfers,
		Flatten:      true,
		MaxDepth:     -1,
	}
	if *got != *want {
		t.Fatalf("buildDownloadCommandOptions() = %#v, want %#v", *got, *want)
	}
}

func TestBuildUploadCommandOptions(t *testing.T) {
	parsed := &transferCLIOptions{recursive: true, flatten: true}
	got := buildUploadCommandOptions(parsed)
	want := &client.UploadOptions{
		Recursive:    true,
		ShowProgress: true,
		Concurrency:  client.MaxConcurrentTransfers,
		Flatten:      true,
		MaxDepth:     -1,
	}
	if *got != *want {
		t.Fatalf("buildUploadCommandOptions() = %#v, want %#v", *got, *want)
	}
}

func TestParseCommandLineBackslashOutsideQuotes(t *testing.T) {
	got := parseCommandLine(`put C:\Users\file.txt`)
	want := []string{"put", `C:\Users\file.txt`}
	if len(got) != len(want) {
		t.Fatalf("parseCommandLine() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseCommandLine()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseCommandLineBackslashInsideDoubleQuotes(t *testing.T) {
	// \P -> \P, \\" -> " (escape consumed, but quote stays open since last char was escaped quote)
	got := parseCommandLine(`put "C:\Program Files\"`)
	// After escaped \" the parser ends with open quote; trailing code writes it
	// Result: C:\Program Files"
	want := []string{"put", "C:\\Program Files\""}
	if len(got) != len(want) {
		t.Fatalf("parseCommandLine() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseCommandLine()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseCommandLineBackslashInsideSingleQuotes(t *testing.T) {
	// Single quotes: backslash is always literal
	got := parseCommandLine(`put 'C:\Users\file.txt'`)
	want := []string{"put", `C:\Users\file.txt`}
	if len(got) != len(want) {
		t.Fatalf("parseCommandLine() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseCommandLine()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseCommandLineEscapedQuoteInsideDoubleQuotes(t *testing.T) {
	got := parseCommandLine(`put "hello \"world"`)
	want := []string{"put", `hello "world`}
	if len(got) != len(want) {
		t.Fatalf("parseCommandLine() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseCommandLine()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseCommandLineEscapedBackslashInsideDoubleQuotes(t *testing.T) {
	got := parseCommandLine(`put "path\\file"`)
	want := []string{"put", `path\file`}
	if len(got) != len(want) {
		t.Fatalf("parseCommandLine() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseCommandLine()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseCommandLineTrailingBackslashOutsideQuotes(t *testing.T) {
	got := parseCommandLine(`put C:\path\`)
	want := []string{"put", `C:\path\`}
	if len(got) != len(want) {
		t.Fatalf("parseCommandLine() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseCommandLine()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
