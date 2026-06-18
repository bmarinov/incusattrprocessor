package cgroup

import (
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkRead measures the syscall cost of reading /proc/<pid>/cgroup
// and the procfs overhead (realistic).
// The temp-file benchmark is for comparison only (page-cached & unrealistic).
func BenchmarkRead(b *testing.B) {
	b.Run("proc-self", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = Read("/proc", "self")
		}
	})

	b.Run("temp-file", func(b *testing.B) {
		dir := b.TempDir()
		pid := "12345"
		if err := os.MkdirAll(filepath.Join(dir, pid), 0o755); err != nil {
			b.Fatal(err)
		}
		const content = "0::/lxc.payload.foo-container\n"
		if err := os.WriteFile(filepath.Join(dir, pid, "cgroup"), []byte(content), 0o644); err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = Read(dir, pid)
		}
	})
}

var sinkName string

// BenchmarkParseLXC measures the string parsing cost.
// Allocations should be zero for the common (LXC) path prefix.
func BenchmarkParseLXC(b *testing.B) {
	const path = "/lxc.payload.foo-container"
	b.ReportAllocs()
	for b.Loop() {
		sinkName, _ = ParseLXC(path)
	}
}
