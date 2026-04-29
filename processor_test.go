package incusattrprocessor

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

func TestIncusAttrProcessor_processProfiles(t *testing.T) {
	tests := []struct {
		name      string
		pid       int64
		instances map[string]InstanceMetadata
		want      *InstanceMetadata
	}{
		{
			name: "enriches resource with container metadata when pid matches a running instance",
			pid:  1122,
			instances: map[string]InstanceMetadata{
				"1122": {Name: "container-foo", Project: "default", Location: "node-0"},
			},
			want: &InstanceMetadata{Name: "container-foo", Project: "default", Location: "node-0"},
		},
		{
			name:      "leaves resource unchanged when pid belongs to no known instance",
			pid:       9001,
			instances: map[string]InstanceMetadata{},
			want:      nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pd, _ := profilesWithPID(tc.pid)
			p := newIncusAttrProcessor(context.Background(), nopSettings(), &processorConfig{},
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

			assertAttr(t, attrs, attrInstanceName, tc.want.Name)
			assertAttr(t, attrs, attrInstanceProject, tc.want.Project)
			assertAttr(t, attrs, attrInstanceLocation, tc.want.Location)
		})
	}
}

func assertAttr(t *testing.T, attrs pcommon.Map, key, want string) {
	t.Helper()
	got, ok := attrs.Get(key)
	if !ok {
		t.Errorf("attribute %s missing", key)
		return
	}
	if got.Str() != want {
		t.Errorf("%s = %q, want %q", key, got.Str(), want)
	}
}

func profilesWithPID(pid int64) (pprofile.Profiles, pprofile.ResourceProfiles) {
	pd := pprofile.NewProfiles()
	rp := pd.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutInt(attrPID, pid)
	return pd, rp
}

func nopSettings() processor.Settings {
	return processor.Settings{
		TelemetrySettings: component.TelemetrySettings{Logger: zap.NewNop()},
	}
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
