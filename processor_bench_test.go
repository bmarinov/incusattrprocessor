package incusattrprocessor

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/bmarinov/otelcol-processor-incus/internal/incus"
	"github.com/bmarinov/otelcol-processor-incus/internal/metadata"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.uber.org/zap"
)

// BenchmarkProcessProfiles measures the hot path through processProfiles.
// Setup: warm Cache, fake InstanceLookup at the Incus boundary (irrelevant).
func BenchmarkProcessProfiles(b *testing.B) {
	type ratio struct {
		name         string
		containerPct int
	}

	batchSizes := []int{1, 10, 100}
	ratios := []ratio{
		{"all-container", 100},
		{"mixed-50", 50},
		{"all-host", 0},
	}

	// Real /proc, host PIDs only. Container PIDs require a live Incus instance.
	b.Run("real proc", func(b *testing.B) {
		const bs = 100
		cache := metadata.NewCache(nil, &noopEventSource{}, warmupWith(), zap.NewNop())
		if err := cache.Start(context.Background()); err != nil {
			b.Fatal(err)
		}
		src := metadata.NewSource(cache, "/proc")
		p := newIncusAttrProcessor(nopSettings(), &processorConfig{}, src, noStart)

		pid := int64(os.Getpid())
		pd := pprofile.NewProfiles()
		for range bs {
			rp := pd.ResourceProfiles().AppendEmpty()
			rp.Resource().Attributes().PutInt(attrPID, pid)
		}

		ctx := context.Background()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = p.processProfiles(ctx, pd)
		}
		b.ReportMetric(float64(bs*b.N)/b.Elapsed().Seconds(), "profiles/s")
	})

	b.Run("page-cached", func(b *testing.B) {
		for _, bs := range batchSizes {
			b.Run(fmt.Sprintf("%d", bs), func(b *testing.B) {
				for _, r := range ratios {
					b.Run(r.name, func(b *testing.B) {
						procRoot := b.TempDir()
						nContainer := (bs * r.containerPct) / 100
						nHost := bs - nContainer

						var seeds []incus.InstanceInfo
						for i := range nContainer {
							pid := fmt.Sprintf("%d", 1000+i)
							name := fmt.Sprintf("container-%d", i)
							writeCgroup(b, procRoot, pid, fmt.Sprintf("0::/lxc.payload.%s\n", name))
							seeds = append(seeds, incus.InstanceInfo{
								Name:     name,
								Project:  "default",
								Location: "node-0",
							})
						}
						for i := range nHost {
							pid := fmt.Sprintf("%d", 2000+i)
							writeCgroup(b, procRoot, pid, "0::/system.slice/sshd.service\n")
						}

						cache := metadata.NewCache(nil, &noopEventSource{}, warmupWith(seeds...), zap.NewNop())
						if err := cache.Start(context.Background()); err != nil {
							b.Fatal(err)
						}
						src := metadata.NewSource(cache, procRoot)
						p := newIncusAttrProcessor(nopSettings(), &processorConfig{}, src, noStart)

						// Reused so allocs/op reflects processProfiles, not buildBatch.
						pd := buildBatch(nContainer, nHost)
						ctx := context.Background()
						b.ReportAllocs()
						b.ResetTimer()
						for i := 0; i < b.N; i++ {
							_, _ = p.processProfiles(ctx, pd)
						}
						b.ReportMetric(float64(bs*b.N)/b.Elapsed().Seconds(), "profiles/s")
					})
				}
			})
		}
	})
}

// buildBatch constructs a Profiles message with
// nContainer PIDs (1000...) and nHost host PIDs (2000...).
// Matches the cgroup files written by the benchmark setup.
func buildBatch(nContainer, nHost int) pprofile.Profiles {
	pd := pprofile.NewProfiles()
	for i := range nContainer {
		rp := pd.ResourceProfiles().AppendEmpty()
		rp.Resource().Attributes().PutInt(attrPID, int64(1000+i))
	}
	for i := range nHost {
		rp := pd.ResourceProfiles().AppendEmpty()
		rp.Resource().Attributes().PutInt(attrPID, int64(2000+i))
	}
	return pd
}
