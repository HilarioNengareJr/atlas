package doctor

import (
	"bufio"
	"os"
	"strings"
)

// envFile is the parsed contents of a .env file: variable name to value.
// A nil envFile is usable — it simply holds nothing, which is what an absent
// .env means for lookup purposes.
type envFile map[string]string

// readEnvFile parses .env at path into name/value pairs.
//
// Deliberately the minimal honest subset, not a dotenv implementation:
// `KEY=value` lines, `#` comments, blank lines skipped, an optional `export `
// prefix tolerated, and matched surrounding quotes stripped. No interpolation
// (`$OTHER`), no multi-line values, no escape sequences. Doctor only ever
// asks "is this name set to something non-empty" — every feature beyond that
// is scope this milestone excluded, and a half-implemented interpolation
// would answer that question WRONG rather than merely incompletely.
//
// An absent .env is not an error: plenty of repos keep their secrets in the
// real environment. It returns an empty envFile, and lookup falls through to
// os.Getenv. Any other read error (a directory named .env, permissions) is
// returned so the caller can surface it rather than silently reporting every
// variable missing.
func readEnvFile(path string) (envFile, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return envFile{}, nil
		}
		return nil, err
	}
	defer f.Close()

	out := envFile{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			continue // not an assignment — nothing to learn from it
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out[name] = unquote(strings.TrimSpace(value))
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// unquote strips one layer of matched surrounding quotes. Unmatched or absent
// quotes are left exactly as they are — guessing at a malformed value would
// be inventing data.
func unquote(s string) string {
	if len(s) < 2 {
		return s
	}
	if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}

// isSet reports whether name resolves to a non-empty value: the repo's .env
// first, then the process environment. Either source satisfies the check —
// .env is where the project's own secrets live, and the process environment
// is how CI and shell-exported secrets arrive.
//
// An empty value counts as NOT set. `KEY=` in a .env file is indistinguishable
// from a half-finished edit, and treating it as present would let doctor pass
// a repo whose MCP will fail on its first call.
func (e envFile) isSet(name string) bool {
	if v, ok := e[name]; ok && v != "" {
		return true
	}
	return os.Getenv(name) != ""
}
