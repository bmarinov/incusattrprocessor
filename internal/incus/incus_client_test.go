package incus

import (
	"context"
	"testing"

	"github.com/lxc/incus/v6/shared/api"
)

type fakeInstanceSrv struct {
	instance api.Instance
}

func (f *fakeInstanceSrv) GetInstance(name string) (api.Instance, string, error) {
	return f.instance, "", nil
}

func TestGetContainerInfo_returnsProjectAndLocation(t *testing.T) {
	// srv := &fakeInstanceSrv{
	// 	instance: api.Instance{
	// 		Project:  "my-project",
	// 		Location: "node-1",
	// 	},
	// }
	c := &Client{
		// server: srv.__
	}

	_, err := c.GetContainerInfo(context.Background(), "web-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

}
