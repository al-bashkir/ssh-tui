package hosts

import (
	"bufio"
	"fmt"
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

// LoadKnownHosts reads hosts from paths, or ~/.ssh/known_hosts when paths is
// empty. Unreadable files are reported in errs and skipped.
func LoadKnownHosts(paths []string) (res LoadResult, errs []error) {
	if len(paths) == 0 {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			paths = []string{filepath.Join(home, ".ssh", "known_hosts")}
		}
	}

	var out []string
	var skipped int

	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}

		// #nosec G304 -- path is user-provided by design (CLI flag / config).
		f, err := os.Open(p)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		hosts, sk, perr := ParseKnownHosts(f)
		_ = f.Close()
		skipped += sk
		if perr != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p, perr))
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
