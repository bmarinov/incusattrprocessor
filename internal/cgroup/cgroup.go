package cgroup

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrNotContainer = errors.New("process not in a container")
)

// Read returns the raw cgroup v2 path.
func Read(procRoot string, pid string) (string, error) {
	f, err := os.Open(filepath.Join(procRoot, pid, "cgroup"))
	if err != nil {
		return "", fmt.Errorf("reading cgroup for pid %s: %w", pid, err)
	}
	defer func() {
		_ = f.Close()
	}()

	cgroupPath, err := parseCgroupFile(f)
	if err != nil {
		return "", fmt.Errorf("pid %s: %w", pid, err)
	}
	return cgroupPath, nil
}

// ParseLXC extracts the instance name from an LXC cgroup path.
func ParseLXC(path string) (string, error) {
	const prefix = "/lxc.payload."
	rest, ok := strings.CutPrefix(path, prefix)
	if !ok {
		return "", fmt.Errorf("not an LXC cgroup path %q: %w", path, ErrNotContainer)
	}

	// Drop subscope
	name, _, _ := strings.Cut(rest, "/")
	if name == "" {
		return "", fmt.Errorf("empty container name in cgroup path %q: %w", path, ErrNotContainer)
	}
	return name, nil
}

// parseCgroupFile extracts the cgroup path from a /proc/<pid>/cgroup file.
func parseCgroupFile(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		path, ok := strings.CutPrefix(line, "0::")
		if ok {
			return path, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scanning cgroup file: %w", err)
	}
	return "", fmt.Errorf("no cgroup v2 unified hierarchy found")
}
