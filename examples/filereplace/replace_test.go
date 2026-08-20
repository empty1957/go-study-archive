package filereplace

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")
	if err := os.WriteFile(path, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Replace(path, []byte("new\n"), 0o600); err != nil {
		t.Fatalf("Replace() error = %v", err)
	}

	assertFileContents(t, path, "new\n")
	assertNoTemporaryFiles(t, dir)
}

func TestReplaceProcessInterruption(t *testing.T) {
	if path := os.Getenv("GO_FILE_REPLACE_HELPER_PATH"); path != "" {
		stopAt := replaceStage(os.Getenv("GO_FILE_REPLACE_STOP_STAGE"))
		err := replace(path, []byte("new\n"), 0o600, func(stage replaceStage) {
			if stage == stopAt {
				os.Exit(23)
			}
		})
		if err != nil {
			os.Exit(24)
		}
		os.Exit(25)
	}

	tests := []struct {
		name    string
		stopAt  replaceStage
		want    string
		wantTmp bool
	}{
		{
			name:    "before rename keeps old target",
			stopAt:  stageTemporaryFileSynced,
			want:    "old\n",
			wantTmp: true,
		},
		{
			name:    "after rename exposes new target",
			stopAt:  stageTargetReplaced,
			want:    "new\n",
			wantTmp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "tasks.json")
			if err := os.WriteFile(path, []byte("old\n"), 0o600); err != nil {
				t.Fatal(err)
			}

			cmd := exec.Command(os.Args[0], "-test.run=^TestReplaceProcessInterruption$")
			cmd.Env = append(os.Environ(),
				"GO_FILE_REPLACE_HELPER_PATH="+path,
				"GO_FILE_REPLACE_STOP_STAGE="+string(tt.stopAt),
			)
			if err := cmd.Run(); err == nil {
				t.Fatal("helper completed; want simulated process interruption")
			} else if exitError, ok := err.(*exec.ExitError); !ok || exitError.ExitCode() != 23 {
				t.Fatalf("helper error = %v, want exit code 23", err)
			}

			assertFileContents(t, path, tt.want)
			temporary, err := filepath.Glob(filepath.Join(dir, ".tasks.json.tmp-*"))
			if err != nil {
				t.Fatal(err)
			}
			if got := len(temporary) > 0; got != tt.wantTmp {
				t.Fatalf("temporary file present = %v, want %v (files: %v)", got, tt.wantTmp, temporary)
			}
		})
	}
}

func TestReplaceRenameFailureKeepsTargetAndCleansUp(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "tasks.json")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}

	if err := Replace(target, []byte("new\n"), 0o600); err == nil {
		t.Fatal("Replace() error = nil, want rename failure")
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("Stat(target) error = %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("target is directory = false, want true")
	}
	assertNoTemporaryFiles(t, dir)
}

func assertFileContents(t *testing.T, path, want string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if got := string(contents); got != want {
		t.Fatalf("ReadFile(%q) = %q, want %q", path, got, want)
	}
}

func assertNoTemporaryFiles(t *testing.T, dir string) {
	t.Helper()
	temporary, err := filepath.Glob(filepath.Join(dir, ".tasks.json.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temporary) != 0 {
		t.Fatalf("temporary files remain: %v", temporary)
	}
}
