package node

import (
	"container/list"
	"fmt"

	"github.com/serverledge-faas/serverledge/internal/container"
	"github.com/serverledge-faas/serverledge/internal/function"
)

func resetPool(f *function.Function) *ContainerPool {
	fp := &ContainerPool{
		busy: list.New(),
		idle: list.New(),
	}
	LocalResources.containerPools[f.Name] = fp
	return fp
}

func injectBusyContainers(fp *ContainerPool, count int) []*container.Container {
	conts := make([]*container.Container, count)
	for i := 0; i < count; i++ {
		c := &container.Container{
			ID:            container.ContainerID(fmt.Sprintf("bench-cont-%d", i)),
			RequestsCount: 1, // so it's busy
		}
		conts[i] = c
		fp.busy.PushBack(c)
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
		fp.idle.PushBack(c)
	}
	return conts
}
