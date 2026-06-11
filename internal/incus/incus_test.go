package incus

import (
	"testing"
)

func TestSplitLabel(t *testing.T) {
	tests := []struct {
		label       string
		wantProject string
		wantName    string
	}{
		{
			label:       "traefik-reverse-proxy",
			wantProject: "default",
			wantName:    "traefik-reverse-proxy",
		},
		{
			label:       "fooproject_sharing-martin",
			wantProject: "fooproject",
			wantName:    "sharing-martin",
		},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			project, name := SplitLabel(tc.label)
			if project != tc.wantProject {
				t.Errorf("project: expected %q, got %q", tc.wantProject, project)
			}
			if name != tc.wantName {
				t.Errorf("name: expected %q, got %q", tc.wantName, name)
			}
		})
	}
}
