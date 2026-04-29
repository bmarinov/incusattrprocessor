package instanceid

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func Test_FromPID_fixtures(t *testing.T) {
	tests := []struct {
		pid     string
		want    string
		wantErr error
	}{
		{pid: "lxc-init-scope", want: "traefik-reverse-proxy"},
		{pid: "lxc-systemd-service", want: "traefik-reverse-proxy"},
		{pid: "lxc-nested-docker", want: "traefik-reverse-proxy"},
		{pid: "host-process", wantErr: ErrNotContainer},
		{pid: "no-such-pid", wantErr: os.ErrNotExist},
	}
	for _, tc := range tests {
		t.Run(tc.pid, func(t *testing.T) {
			name, err := FromPID("testdata/proc", tc.pid)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if name != tc.want {
					t.Errorf("expected name %s, got %s", tc.want, name)
				}
			}
		})
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

func Test_parseLXCCgroupPath(t *testing.T) {
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
			name, err := parseLXCCgroupPath(tc.path)
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
