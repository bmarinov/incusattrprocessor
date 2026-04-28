package incusattrprocessor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_parseCgroupFile(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "cgroupv2 unified hierarchy",
			input: "0::/lxc.payload.mycontainer\n",
			want:  "/lxc.payload.mycontainer",
		},
		{
			name:  "cgroupv2 with scope subpath",
			input: "0::/lxc.payload.mycontainer/init.scope\n",
			want:  "/lxc.payload.mycontainer/init.scope",
		},
		{
			name: "cgroupv1 controllers plus cgroupv2 unified",
			input: "12:devices:/lxc.payload.mycontainer\n" +
				"11:memory:/lxc.payload.mycontainer\n" +
				"0::/lxc.payload.mycontainer\n",
			want: "/lxc.payload.mycontainer",
		},
		{
			name:  "process not in a container",
			input: "0::/system.slice/sshd.service\n",
			want:  "/system.slice/sshd.service",
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
				t.Fatalf("parseCgroupFile() error = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("parseCgroupFile() = %s, want %s", got, tc.want)
			}
		})
	}
}

func Test_parseIncusCgroupPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantName string
		wantErr  bool
	}{
		{
			name:     "container name only",
			path:     "/lxc.payload.mycontainer",
			wantName: "mycontainer",
		},
		{
			name:     "scope subpath is stripped",
			path:     "/lxc.payload.mycontainer/init.scope",
			wantName: "mycontainer",
		},
		{
			name:    "not an incus path",
			path:    "/system.slice/docker.service",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name, err := parseIncusCgroupPath(tc.path)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseIncusCgroupPath() error = %v, wantErr %v", err, tc.wantErr)
			}
			if name != tc.wantName {
				t.Errorf("name = %s, want %s", name, tc.wantName)
			}
		})
	}
}

func TestCgroupMetadataSource_GetInstanceMetadata(t *testing.T) {
	procRoot := t.TempDir()

	writeProc := func(t *testing.T, pid, cgroupContent string) {
		t.Helper()
		dir := filepath.Join(procRoot, pid)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "cgroup"), []byte(cgroupContent), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name       string
		pid        string
		cgroupFile string
		wantName   string
		wantErr    bool
	}{
		{
			name:       "resolves container name from cgroup",
			pid:        "100",
			cgroupFile: "0::/lxc.payload.web-frontend\n",
			wantName:   "web-frontend",
		},
		{
			name:       "scope subpath does not affect name",
			pid:        "200",
			cgroupFile: "0::/lxc.payload.api-server/init.scope\n",
			wantName:   "api-server",
		},
		{
			name:       "pid not in a container",
			pid:        "300",
			cgroupFile: "0::/system.slice/sshd.service\n",
			wantErr:    true,
		},
		{
			name:    "pid does not exist",
			pid:     "999",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.cgroupFile != "" {
				writeProc(t, tc.pid, tc.cgroupFile)
			}
			src := &cgroupMetadataSource{procRoot: procRoot}

			got, err := src.GetInstanceMetadata(t.Context(), tc.pid)
			if (err != nil) != tc.wantErr {
				t.Fatalf("GetInstanceMetadata() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if got.Name != tc.wantName {
				t.Errorf("Name = %s, want %s", got.Name, tc.wantName)
			}
		})
	}
}
