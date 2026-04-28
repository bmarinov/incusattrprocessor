package incusattrprocessor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type cgroupMetadataSource struct {
	procRoot string // e.g. "/proc"
}

func newCgroupMetadataSource() *cgroupMetadataSource {
	return &cgroupMetadataSource{procRoot: "/proc"}
}

func (s *cgroupMetadataSource) GetInstanceMetadata(_ context.Context, pid string) (InstanceMetadata, error) {
	f, err := os.Open(filepath.Join(s.procRoot, pid, "cgroup"))
	if err != nil {
		return InstanceMetadata{}, fmt.Errorf("reading cgroup for pid %s: %w", pid, err)
	}
	defer f.Close()

	cgroupPath, err := parseCgroupFile(f)
	if err != nil {
		return InstanceMetadata{}, fmt.Errorf("pid %s: %w", pid, err)
	}

	name, err := parseIncusCgroupPath(cgroupPath)
	if err != nil {
		return InstanceMetadata{}, fmt.Errorf("pid %s: %w", pid, err)
	}

	return InstanceMetadata{Name: name}, nil
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
	return "", fmt.Errorf("no cgroup v2 unified hierarchy entry found")
}

// parseIncusCgroupPath extracts the instance name from an Incus cgroup path.
// Expected format: /lxc.payload.<name>[/<subpath>]
func parseIncusCgroupPath(path string) (string, error) {
	const prefix = "/lxc.payload."
	rest, ok := strings.CutPrefix(path, prefix)
	if !ok {
		return "", fmt.Errorf("not an Incus cgroup path: %q", path)
	}

	// Drop subscope
	name, _, _ := strings.Cut(rest, "/")
	return name, nil
}
