package incusattrprocessor

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

func TestIncusAttrProcessor_processProfiles(t *testing.T) {
	tests := []struct {
		name      string
		pid       string
		instances map[string]InstanceMetadata
		want      *InstanceMetadata
	}{
		{
			name: "adds instance metadata when pid matches",
			pid:  "1122",
			instances: map[string]InstanceMetadata{
				"1122": {Name: "container-foo", Project: "default", Location: "node-0"},
			},
			want: &InstanceMetadata{Name: "container-foo", Project: "default", Location: "node-0"},
		},
		{
			name:      "leaves resource unchanged when pid has no matching instance",
			pid:       "9001",
			instances: map[string]InstanceMetadata{},
			want:      nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pd, _ := profilesWithPID(tc.pid)
			p := newIncusAttrProcessor(context.Background(), &processorConfig{},
				&metadataSourceFake{instances: tc.instances})

			got, err := p.processProfiles(context.Background(), pd)
			if err != nil {
				t.Fatalf("processProfiles returned unexpected error: %v", err)
			}
			attrs := got.ResourceProfiles().At(0).Resource().Attributes()

			if tc.want == nil {
				if _, ok := attrs.Get(attrInstanceName); ok {
					t.Errorf("expected no %s attribute, but got one", attrInstanceName)
				}
				return
			}

			assertAttrs(t, attrs, attrInstanceName, tc.want.Name)
			assertAttrs(t, attrs, attrInstanceProject, tc.want.Project)
			assertAttrs(t, attrs, attrInstanceLocation, tc.want.Location)
		})
	}
}

func assertAttrs(t *testing.T, attrs pcommon.Map, key, want string) {
	t.Helper()
	got, ok := attrs.Get(key)
	if !ok {
		t.Errorf("attribute %s missing", key)
		return
	}
	if got.Str() != want {
		t.Errorf("%s = %s, want %s", key, got.Str(), want)
	}
}

func profilesWithPID(pid string) (pprofile.Profiles, pprofile.ResourceProfiles) {
	pd := pprofile.NewProfiles()
	rp := pd.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutStr("process.pid", pid)
	return pd, rp
}

type metadataSourceFake struct {
	instances map[string]InstanceMetadata
}

func (f *metadataSourceFake) GetInstanceMetadata(_ context.Context, id string) (InstanceMetadata, error) {
	m, ok := f.instances[id]
	if !ok {
		return InstanceMetadata{}, errors.New("pid not found")
	}
	return m, nil
}
