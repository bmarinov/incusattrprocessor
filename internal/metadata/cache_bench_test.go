package metadata

import (
	"context"
	"fmt"
	"testing"

	"github.com/bmarinov/incusattrprocessor/internal/incus"
	"go.uber.org/zap"
)

func BenchmarkCacheGetInstance(b *testing.B) {
	seed := incus.InstanceInfo{
		Name:         "web-fe-123",
		Project:      "default",
		Location:     "node-0",
		Architecture: "amd64",
	}

	newWarmedCache := func(b *testing.B) *Cache {
		b.Helper()
		c := NewCache(
			&fakeInstanceLookup{},
			&fakeEventSource{},
			func(_ context.Context) ([]incus.InstanceInfo, error) {
				return []incus.InstanceInfo{seed}, nil
			},
			zap.NewNop(),
		)
		if err := c.Start(context.Background()); err != nil {
			b.Fatal(err)
		}
		return c
	}

	b.Run("warm hit", func(b *testing.B) {
		c := newWarmedCache(b)
		ctx := context.Background()
		b.ReportAllocs()
		for b.Loop() {
			_, _ = c.GetInstance(ctx, seed.Project, seed.Name)
		}
	})

	b.Run("warm hit parallel", func(b *testing.B) {
		c := newWarmedCache(b)
		ctx := context.Background()
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = c.GetInstance(ctx, seed.Project, seed.Name)
			}
		})
	})

	// miss: RLock + map miss + goroutine + partial return.
	// Worst case during cache warmup, not representative.
	b.Run("miss", func(b *testing.B) {
		c := NewCache(
			&fakeInstanceLookup{info: incus.InstanceInfo{Location: "node-0", Architecture: "amd64"}},
			&fakeEventSource{},
			func(_ context.Context) ([]incus.InstanceInfo, error) { return nil, nil },
			zap.NewNop(),
		)
		if err := c.Start(context.Background()); err != nil {
			b.Fatal(err)
		}

		keys := make([]string, b.N)
		for i := range keys {
			keys[i] = fmt.Sprintf("container-%d", i)
		}

		ctx := context.Background()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = c.GetInstance(ctx, "default", keys[i])
		}
	})
}
