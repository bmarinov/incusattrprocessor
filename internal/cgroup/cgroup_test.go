package cgroup

import (
	"bufio"
	"errors"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRead(t *testing.T) {
	t.Run("no such path", func(t *testing.T) {
		path, err := Read("foo/proc", "123")
		if err == nil {
			t.Fatalf("expected err, got result %v", path)
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("expected %v got %v", os.ErrNotExist, err)
		}
	})
}

func TestParseLXC(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantName string
		wantErr  error
	}{
		{
			name:     "container name only",
			path:     "/lxc.payload.foo-container",
			wantName: "foo-container",
		},
		{
			name:     "scope subpath is stripped",
			path:     "/lxc.payload.foo-container/init.scope",
			wantName: "foo-container",
		},
		{
			name:    "empty name after prefix",
			path:    "/lxc.payload./init.scope",
			wantErr: ErrNotContainer,
		},
		{
			name:    "not an LXC path",
			path:    "/system.slice/docker.service",
			wantErr: ErrNotContainer,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name, err := ParseLXC(tc.path)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected err %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if name != tc.wantName {
				t.Errorf("expected name %s, got %s", tc.wantName, name)
			}
		})
	}
}

func TestRead_Parse_Fixtures(t *testing.T) {
	for caseName, expected := range testCases(t, filepath.Join("testdata/proc", "oracle")) {
		t.Run(caseName, func(t *testing.T) {
			cgroupPath, err := Read("testdata/proc", caseName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, err := ParseLXC(cgroupPath)
			if strings.HasPrefix(expected, "!") {
				sentinel := resolveErrorSentinel(t, expected[1:])
				if !errors.Is(err, sentinel) {
					t.Fatalf("got error %v, want %v", err, sentinel)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if got != expected {
				t.Errorf("expected %q, got %q", expected, got)
			}
		})
	}
}

func testCases(t *testing.T, testOracle string) iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		f, err := os.Open(testOracle)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = f.Close()
		}()

		scanner := bufio.NewScanner(f)
		defer func() {
			if scanner.Err() != nil {
				t.Errorf("test oracle scanner error: %v", scanner.Err())
			}
		}()

		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)
			if len(fields) != 2 {
				t.Errorf("expected two fields, got %d - line %q", len(fields), line)
				continue
			}
			caseName, expected := fields[0], fields[1]
			if !yield(caseName, expected) {
				return
			}
		}
	}
}

func resolveErrorSentinel(t *testing.T, name string) error {
	t.Helper()
	switch name {
	case "ErrNotContainer":
		return ErrNotContainer
	default:
		t.Fatalf("unknown error sentinel %q in oracle", name)
		return nil
	}
}

func Test_parseCgroupFile(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "cgroupv2 lxc container",
			input: "0::/lxc.payload.foo-container\n",
			want:  "/lxc.payload.foo-container",
		},
		{
			name:  "cgroupv2 with scope subpath",
			input: "0::/lxc.payload.foo-container/init.scope\n",
			want:  "/lxc.payload.foo-container/init.scope",
		},
		{
			name:  "non-LXC path",
			input: "0::/system.slice/sshd.service\n",
			want:  "/system.slice/sshd.service",
		},
		{
			name:    "no cgroupv2 entry",
			input:   "12:devices:/lxc.payload.baz-container\n11:memory:/lxc.payload.baz-container\n",
			wantErr: true,
		},
		{
			name:    "empty file",
			input:   "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseCgroupFile(strings.NewReader(tc.input))
			if (err != nil) != tc.wantErr {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}
			if got != tc.want {
				t.Errorf("expected %s, got %s", tc.want, got)
			}
		})
	}
}
