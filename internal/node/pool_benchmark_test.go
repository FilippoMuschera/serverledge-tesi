package node

import (
	"fmt"
	"testing"

	"github.com/serverledge-faas/serverledge/internal/function"
)

func init() {
	if LocalResources.containerPools == nil {
		LocalResources.containerPools = make(map[string]*ContainerPool)
	}
	// Mock resources to simulate the allocation of many containers without problems
	LocalResources.usedCPUs = 0

}

// benchmark moving a container from busy to idle
func BenchmarkPoolCycle(b *testing.B) {
	f := &function.Function{Name: "bench_cycle", MemoryMB: 1, CPUDemand: 0.0000000000001}

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
