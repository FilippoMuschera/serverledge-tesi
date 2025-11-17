package lb

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/serverledge-faas/serverledge/internal/config"
	"github.com/serverledge-faas/serverledge/internal/container"
	"github.com/serverledge-faas/serverledge/internal/function"
)

type ArchitectureAwareBalancer struct {
	mu sync.Mutex

	// instead of classic lists we will use hashRings (see hashRing.go) to implement a consistent hashing technique
	armRing *HashRing
	x86Ring *HashRing
}

// NewArchitectureAwareBalancer Constructor
func NewArchitectureAwareBalancer(targets []*middleware.ProxyTarget) *ArchitectureAwareBalancer {

	// REPLICAS is the number of times each physical node will appear in the hash ring. This is done to improve how
	// virtual nodes (i.e.: replicas of each physical node) are distributed over the ring, to reduce variation.
	REPLICAS := config.GetInt(config.REPLICAS, 16)
	log.Printf("Running ArchitectureAwareLB with %d replicas per node in the hash rings\n", REPLICAS)

	b := &ArchitectureAwareBalancer{
		armRing: NewHashRing(REPLICAS),
		x86Ring: NewHashRing(REPLICAS),
	}

	// to stay consistent with the old RoundRobinLoadBalancer, we'll still a single target list, that will contain all nodes,
	// both ARM and x86. We will now sort them into the respective hashRings.
	for _, t := range targets {
		arch := t.Meta["arch"]
		if arch == container.ARM {
			b.armRing.Add(t)
		} else if arch == container.X86 {
			b.x86Ring.Add(t)
		} else {
			log.Printf("Unknown architecture for node %s\n", t.Name)
		}
	}

	return b
}

// Next Used by Echo Proxy middleware to select the next target dynamically
func (b *ArchitectureAwareBalancer) Next(c echo.Context) *middleware.ProxyTarget {
	b.mu.Lock()
	defer b.mu.Unlock()

	funcName := extractFunctionName(c)        // get function's name from request's URL
	fun, ok := function.GetFunction(funcName) // we use this to leverage cache before asking etcd
	if !ok {
		log.Printf("Dropping request for unknown fun '%s'\n", funcName)
		return nil
	}

	targetArch, err := b.selectArchitecture(fun) // here the load balancer decides what architecture to use for this function
	if err != nil {
		log.Printf("Failed to select a target for function '%s': %v", funcName, err)
		return nil // No suitable node found
	}

	// once we selected an architecture, we'll use consistent hashing to select what node to use
	// The Get function will cycle through the hashRing to find a suitable node. If none is find we try to check if in
	// the other ring there is a suitable node for the function, to maximize chances of execution.
	var candidate *middleware.ProxyTarget
	if targetArch == container.ARM { // Prioritize ARM if selected
		candidate = b.armRing.Get(fun)
		if candidate == nil && fun.SupportsArch(container.X86) { // If no ARM node, try x86 if supported
			candidate = b.x86Ring.Get(fun)
		}
	} else { // Prioritize x86 if selected
		candidate = b.x86Ring.Get(fun)
		if candidate == nil && fun.SupportsArch(container.ARM) { // If no x86 node, try ARM if supported
			candidate = b.armRing.Get(fun)
		}
	}
	return candidate
}

// extractFunctionName retrieves the function's name by parsing the request's URL.
func extractFunctionName(c echo.Context) string {
	path := c.Request().URL.Path

	const prefix = "/invoke/"
	if !strings.HasPrefix(path, prefix) {
		return "" // not an invocation
	}

	return path[len(prefix):]
}

// selectArchitecture checks the function's runtime to see what architecture it can support. Then it checks if any
// available node of the corresponding architecture is available. If the runtime supports both architecture, then we
// have a tie-break and select a node from the chosen list (arm or x86).
func (b *ArchitectureAwareBalancer) selectArchitecture(fun *function.Function) (string, error) {
	supportsArm := fun.SupportsArch(container.ARM)
	supportsX86 := fun.SupportsArch(container.X86)

	//TODO implement a better tie-breaking strategy

	// Tie-breaking: if both architectures are supported, prefer ARM if available (less energy consumption), otherwise x86.
	if supportsArm && supportsX86 {
		if b.armRing.Size() > 0 {
			return container.ARM, nil
		}
		if b.x86Ring.Size() > 0 {
			return container.X86, nil
		}
		return "", fmt.Errorf("no available nodes for either ARM or x86")
	}

	if supportsArm {
		if b.armRing.Size() > 0 {
			return container.ARM, nil
		}
		return "", fmt.Errorf("no ARM nodes available")
	}

	if supportsX86 {
		if b.x86Ring.Size() > 0 {
			return container.X86, nil
		}
		return "", fmt.Errorf("no x86 nodes available")
	}

	return "", fmt.Errorf("function does not support any available architecture")
}

// AddTarget Echo requires this method for dynamic load-balancing. It simply inserts a new node in the respective ring.
func (b *ArchitectureAwareBalancer) AddTarget(t *middleware.ProxyTarget) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Decide if target belongs to ARM or x86
	if t.Meta["arch"] == container.ARM {
		b.armRing.Add(t)
	} else {
		b.x86Ring.Add(t)
	}

	return true
}

// RemoveTarget Echo requires this method to remove a target by name
func (b *ArchitectureAwareBalancer) RemoveTarget(name string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.armRing.RemoveByName(name) {
		return true
	}
	if b.x86Ring.RemoveByName(name) {
		return true
	}
	return false

}
