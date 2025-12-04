package node

import (
	"fmt"

	"github.com/serverledge-faas/serverledge/internal/container"
	"github.com/serverledge-faas/serverledge/internal/function"
)

// resetPool specific for slice implementation
func resetPool(f *function.Function) *ContainerPool {
	fp := &ContainerPool{
		busy: make([]*container.Container, 0),
		idle: make([]*container.Container, 0),
	}
	LocalResources.containerPools[f.Name] = fp
	return fp
}

func injectBusyContainers(fp *ContainerPool, count int) []*container.Container {
	conts := make([]*container.Container, count)
	for i := 0; i < count; i++ {
		c := &container.Container{
			ID:            container.ContainerID(fmt.Sprintf("bench-cont-%d", i)),
			RequestsCount: 1,
		}
		conts[i] = c
		fp.busy = append(fp.busy, c)
	}
	return conts
}

func injectIdleContainers(fp *ContainerPool, count int) []*container.Container {
	conts := make([]*container.Container, count)
	for i := 0; i < count; i++ {
		c := &container.Container{
			ID: container.ContainerID(fmt.Sprintf("idle-cont-%d", i)),
		}
		conts[i] = c
		fp.idle = append(fp.idle, c)
	}
	return conts
}
