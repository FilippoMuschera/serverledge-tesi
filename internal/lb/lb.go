package lb

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/serverledge-faas/serverledge/internal/config"
	"github.com/serverledge-faas/serverledge/internal/registration"
)

var currentTargets []*middleware.ProxyTarget

func newBalancer(targets []*middleware.ProxyTarget) middleware.ProxyBalancer {
	// old Load Balancer: return middleware.NewRoundRobinBalancer(targets)
	return NewArchitectureAwareBalancer(targets)
}

func StartReverseProxy(e *echo.Echo, region string) {
	targets, err := getTargets(region)
	if err != nil {
		log.Printf("Cannot connect to registry to retrieve targets: %v\n", err)
		os.Exit(2)
	}

	log.Printf("Initializing with %d targets.\n", len(targets))
	balancer := newBalancer(targets)
	currentTargets = targets

	// Custom ProxyConfig to process custom headers and update available memory of each targets after they
	// executed a function.
	// These headers are set after the execution of the function on the target node, so the free memory already
	// includes the memory freed by the function, once it's executed.
	proxyConfig := middleware.ProxyConfig{
		Balancer: balancer,

		// We use ModifyResponse to process these headers
		ModifyResponse: func(res *http.Response) error {

			nodeName := res.Header.Get("Serverledge-Node-Name")
			freeMemStr := res.Header.Get("Serverledge-Free-Mem")

			if nodeName != "" && freeMemStr != "" {
				freeMem, err := strconv.ParseInt(freeMemStr, 10, 64)
				if err == nil {
					NodeMetrics.Update(nodeName, freeMem, time.Now().Unix())

					log.Printf("[LB-Update] Node %s reported %d MB free", nodeName, freeMem)
				}
			}

			// Remove the no-longer-needed headers
			res.Header.Del("Serverledge-Node-Name")
			res.Header.Del("Serverledge-Free-Mem")

			return nil
		},
	}

	e.Use(middleware.ProxyWithConfig(proxyConfig))
	go updateTargets(balancer, region)

	portNumber := config.GetInt(config.API_PORT, 1323)
	if err := e.Start(fmt.Sprintf(":%d", portNumber)); err != nil && !errors.Is(err, http.ErrServerClosed) {
		e.Logger.Fatal("shutting down the server")
	}
}

func getTargets(region string) ([]*middleware.ProxyTarget, error) {
	cloudNodes, err := registration.GetNodesInArea(region, false, 0)
	if err != nil {
		return nil, err
	}

	targets := make([]*middleware.ProxyTarget, 0, len(cloudNodes))
	for _, target := range cloudNodes {
		log.Printf("Found target: %v\n", target.Key)
		// TODO: etcd should NOT contain URLs, but only host and port...
		parsedUrl, err := url.Parse(target.APIUrl())
		if err != nil {
			return nil, err
		}
		archMap := echo.Map{"arch": target.Arch}
		targets = append(targets, &middleware.ProxyTarget{Name: target.Key, URL: parsedUrl, Meta: archMap})
	}

	log.Printf("Found %d targets\n", len(targets))

	return targets, nil
}

func updateTargets(balancer middleware.ProxyBalancer, region string) {
	for {
		time.Sleep(30 * time.Second) // TODO: configure

		targets, err := getTargets(region)
		if err != nil {
			log.Printf("Cannot update targets: %v\n", err)
			continue // otherwise we update everything with a nil target array, removing all targets from the LB list!
		}

		toKeep := make([]bool, len(currentTargets))
		for i := range currentTargets {
			toKeep[i] = false
		}
		for _, t := range targets {
			toAdd := true
			for i, curr := range currentTargets {
				if curr.Name == t.Name {
					toKeep[i] = true
					toAdd = false
				}
			}
			if toAdd {
				log.Printf("Adding %s\n", t.Name)
				balancer.AddTarget(t)
			}
		}

		toRemove := make([]string, 0)
		for i, curr := range currentTargets {
			if !toKeep[i] {
				log.Printf("Removing %s\n", curr.Name)
				toRemove = append(toRemove, curr.Name)
			} else {
				// If we keep this node, then we'll update its info about free memory
				nodeInfo := registration.GetSingleNeighborInfo(curr.Name)
				if nodeInfo != nil {
					freeMemoryMB := nodeInfo.TotalMemory - nodeInfo.UsedMemory
					NodeMetrics.Update(curr.Name, freeMemoryMB, nodeInfo.LastUpdateTime)
				}
			}
		}
		for _, curr := range toRemove {
			balancer.RemoveTarget(curr)
		}

		currentTargets = targets
	}
}
