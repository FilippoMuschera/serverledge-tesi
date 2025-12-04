package node

import (
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"github.com/serverledge-faas/serverledge/internal/function"
)

func init() {

	log.SetOutput(io.Discard) // otherwise too many log messages and the outcome of the benchmark is lost inside the logs

	if LocalResources.containerPools == nil {
		LocalResources.containerPools = make(map[string]*ContainerPool)
	}
	// Mock resources to simulate the allocation of many containers without problems
	LocalResources.usedCPUs = 0
	LocalResources.totalCPUs = 512
	LocalResources.totalMemory = 20480

}

// benchmark moving a container from busy to idle
func BenchmarkPoolCycle(b *testing.B) {
	f := &function.Function{Name: "bench_cycle", MemoryMB: 1, CPUDemand: 0.0001}

	// Scenarios:
	// 1: Empty pool (only 1 container running) -> Best case for lists
	// 1000/5000: Full pool (1 container running, 999 interfering) -> Worst case for lists
	poolSizes := []int{1, 100, 1000, 5000}

	for _, size := range poolSizes {
		testName := fmt.Sprintf("PoolSize-%d", size)
		b.Run(testName, func(b *testing.B) {
			b.StopTimer()
			fp := resetPool(f)

			injectBusyContainers(fp, size-1)

			// now the container we'll try to acquire and move to busy
			_ = injectIdleContainers(fp, 1)

			b.StartTimer()

			for i := 0; i < b.N; i++ {

				c, err := acquireWarmContainer(f)
				if err != nil {
					b.Fatalf("Acquire failed: %v", err)
				}

				HandleCompletion(c, f)
			}
		})
	}
}

func BenchmarkJanitorScan(b *testing.B) {
	f := &function.Function{Name: "bench_janitor", MemoryMB: 1, CPUDemand: 0}
	poolSizes := []int{100, 1000, 5000}

	for _, size := range poolSizes {
		testName := fmt.Sprintf("IdleSize-%d", size)
		b.Run(testName, func(b *testing.B) {
			b.StopTimer()
			fp := resetPool(f)

			// Iniettiamo container
			conts := injectIdleContainers(fp, size)

			// we don't want to actually destroy the containers (more overhead by docker + we'd need to actually create them)
			// we simply want to see how fast it is to iterate over the whole lis/slice
			future := time.Now().Add(1 * time.Hour).UnixNano()
			for _, c := range conts {
				c.ExpirationTime = future
			}

			b.StartTimer()

			for i := 0; i < b.N; i++ {
				DeleteExpiredContainer()
			}
		})
	}
}
