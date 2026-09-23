package hosts

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type LoadResult struct {
	Hosts        []string
	SkippedLines int
}

type PathError struct {
	Path string
	Err  error
}

func (e PathError) Error() string {
	if e.Path == "" {
		return e.Err.Error()
	}
	return e.Path + ": " + e.Err.Error()
}

func DefaultKnownHostsPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	return []string{filepath.Join(home, ".ssh", "known_hosts")}
}

func LoadKnownHosts(paths []string) (LoadResult, []PathError) {
	if len(paths) == 0 {
		paths = DefaultKnownHostsPaths()
	}

	var out []string
	var skipped int
	var errs []PathError

	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}

		// #nosec G304 -- path is user-provided by design (CLI flag / config).
		f, err := os.Open(p)
		if err != nil {
			errs = append(errs, PathError{Path: p, Err: err})
			continue
		}

		hosts, sk, perr := ParseKnownHosts(f)
		_ = f.Close()
		skipped += sk
		if perr != nil {
			errs = append(errs, PathError{Path: p, Err: perr})
			continue
		}
		out = append(out, hosts...)
	}

	return LoadResult{Hosts: sortedUnique(out), SkippedLines: skipped}, errs
}

func ParseKnownHosts(r io.Reader) ([]string, int, error) {
	s := bufio.NewScanner(r)
	// known_hosts lines can be long because of key material; bump buffer.
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var hosts []string
	skipped := 0

	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		// Blank lines, comments, markers (@cert-authority/@revoked) and
		// hashed hostnames are not displayable.
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "@") || strings.HasPrefix(line, "|1|") {
			skipped++
			continue
		}

		first := strings.Fields(line)[0]
		for _, raw := range strings.Split(first, ",") {
			h := strings.TrimSpace(raw)
			if h == "" || strings.HasPrefix(h, "|1|") {
				continue
			}
			hosts = append(hosts, h)
		}
	}

	if err := s.Err(); err != nil {
		return nil, skipped, err
	}
	return sortedUnique(hosts), skipped, nil
}

func sortedUnique(s []string) []string {
	slices.Sort(s)
	return slices.Compact(s)
}
