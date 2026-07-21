package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

// writeEnv writes a real .env file into a fresh temp directory and returns
// its path. Real files on disk, not a mock reader — the standard M4.1 set.
func writeEnv(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	return path
}

func TestReadEnvFileParsesTheSupportedSubset(t *testing.T) {
	env, err := readEnvFile(writeEnv(t, `# a comment

PLAIN=value
QUOTED="double quoted"
SINGLE='single quoted'
export EXPORTED=exported-value
SPACED  =  padded
EMPTY=
NO_EQUALS_SIGN
WITH_EQUALS=a=b=c
`))
	if err != nil {
		t.Fatalf("readEnvFile: %v", err)
	}

	want := map[string]string{
		"PLAIN":       "value",
		"QUOTED":      "double quoted",
		"SINGLE":      "single quoted",
		"EXPORTED":    "exported-value",
		"SPACED":      "padded",
		"EMPTY":       "",
		"WITH_EQUALS": "a=b=c",
	}
	for name, wantValue := range want {
		got, ok := env[name]
		if !ok {
			t.Errorf("%s missing from parsed .env", name)
			continue
		}
		if got != wantValue {
			t.Errorf("%s = %q, want %q", name, got, wantValue)
		}
	}
	if _, ok := env["NO_EQUALS_SIGN"]; ok {
		t.Error("a line with no '=' should not become a variable")
	}
}

func TestReadEnvFileAbsentIsNotAnError(t *testing.T) {
	env, err := readEnvFile(filepath.Join(t.TempDir(), ".env"))
	if err != nil {
		t.Fatalf("an absent .env must not be an error, got: %v", err)
	}
	if len(env) != 0 {
		t.Fatalf("absent .env should parse to nothing, got %d entries", len(env))
	}
}

func TestReadEnvFileUnreadableIsAnError(t *testing.T) {
	// A directory named .env is the portable way to make the open fail
	// without depending on running as an unprivileged user.
	dir := filepath.Join(t.TempDir(), ".env")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := readEnvFile(dir); err == nil {
		t.Fatal("a .env that cannot be read as a file must surface an error, not silently report every variable missing")
	}
}

func TestIsSetPrefersEnvFileThenFallsBackToProcess(t *testing.T) {
	env := envFile{"FROM_FILE": "x", "EMPTY_IN_FILE": ""}

	if !env.isSet("FROM_FILE") {
		t.Error("a non-empty value in .env must count as set")
	}
	if env.isSet("NOT_ANYWHERE") {
		t.Error("a name in neither .env nor the process environment must count as unset")
	}

	t.Setenv("FROM_PROCESS", "y")
	if !env.isSet("FROM_PROCESS") {
		t.Error("a name set in the process environment must count as set — CI and exported shell secrets arrive that way")
	}

	// An empty value in .env is a half-finished edit, not a secret. It must
	// not satisfy the check, but it must still fall through to the process
	// environment rather than short-circuiting to false.
	if env.isSet("EMPTY_IN_FILE") {
		t.Error("an empty value in .env must count as unset")
	}
	t.Setenv("EMPTY_IN_FILE", "real-value")
	if !env.isSet("EMPTY_IN_FILE") {
		t.Error("an empty .env value must fall through to the process environment, not short-circuit to unset")
	}
}
