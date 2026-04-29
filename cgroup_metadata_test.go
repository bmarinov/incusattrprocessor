package incusattrprocessor

import (
	"os"
	"path/filepath"
	"testing"
)

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
