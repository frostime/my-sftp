package shell

import "testing"

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
