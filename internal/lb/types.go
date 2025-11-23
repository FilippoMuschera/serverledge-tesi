package lb

import (
	"github.com/labstack/echo/v4/middleware"
	"github.com/serverledge-faas/serverledge/internal/function"
	"github.com/serverledge-faas/serverledge/internal/registration"
)

// MemoryChecker is the function that checks if the node selected has enough memory to execute the function.
// it is an interface, and it's put in HashRing to make unit-tests possible by mocking it
type MemoryChecker interface {
	HasEnoughMemory(target *middleware.ProxyTarget, fun *function.Function) bool
}

type DefaultMemoryChecker struct{}

func (m *DefaultMemoryChecker) HasEnoughMemory(candidate *middleware.ProxyTarget, fun *function.Function) bool {
	nodesInfo := registration.GetFullNeighborInfo()
	if nodesInfo == nil {
		return true // if for some reason I have no information on neighbors, then let's just use the first I found.
		// It still has more chances of a warm start, and I cannot gather information about any node anyway
	}
	candidateInfo := nodesInfo[candidate.Name] // candidate.Name = NodeRegistration.Key, see lb.go
	if candidateInfo == nil {
		return true // not enough information to justify skipping this node
	}

	// UsedMemory refers only to memory used by *running* functions, not memory for warm containers.
	return candidateInfo.TotalMemory-candidateInfo.UsedMemory >= fun.MemoryMB // true if there is sufficient memory for execution

}
