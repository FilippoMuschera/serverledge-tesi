package scheduling

import (
	"github.com/serverledge-faas/serverledge/internal/function"
	"github.com/serverledge-faas/serverledge/internal/node"
)

// EdgePolicy supports only Edge-Edge offloading. Always does offloading to an edge node if enabled. When offloading is not enabled executes the request locally.
type EdgePolicy struct{}

func (p *EdgePolicy) Init() {
}

func (p *EdgePolicy) OnCompletion(_ *function.Function, _ *function.ExecutionReport) {

}

func (p *EdgePolicy) OnArrival(r *scheduledRequest) {
	if r.CanDoOffloading {
		url := pickEdgeNodeForOffloading(r) // this will now take into account the node architecture in the offloading process
		if url != "" {
			handleOffload(r, url)
			return
		}
	} else {

		if !r.Fun.SupportsArch(node.LocalNode.Arch) {
			// If the current node architecture is not supported by the function's runtime, we can only drop it, since
			// offloading was already tried unsuccessfully, or it was disabled for this request.
			dropRequest(r)
			return

		}

		containerID, warm, err := node.AcquireContainer(r.Fun, false)
		if err == nil {
			execLocally(r, containerID, warm)
			return
		}
	}

	dropRequest(r)
}
