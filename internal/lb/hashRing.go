package lb

import (
	"fmt"
	"hash/fnv"
	"log"
	"sort"

	"github.com/labstack/echo/v4/middleware"
)

type HashRing struct {
	replicas int
	ring     []uint32
	targets  map[uint32]*middleware.ProxyTarget
}

func NewHashRing(replicas int) *HashRing {
	return &HashRing{
		replicas: replicas,
		ring:     make([]uint32, 0),
		targets:  make(map[uint32]*middleware.ProxyTarget),
	}
}

func (r *HashRing) Add(t *middleware.ProxyTarget) {
	for i := 0; i < r.replicas; i++ {
		key := fmt.Sprintf("%s#%d", t.Name, i)
		h := hash(key)
		r.ring = append(r.ring, h)
		r.targets[h] = t
	}
	sort.Slice(r.ring, func(i, j int) bool { return r.ring[i] < r.ring[j] })
}

func (r *HashRing) Get(key string) *middleware.ProxyTarget {
	if len(r.ring) == 0 {
		return nil
	}

	h := hash(key)
	idx := sort.Search(len(r.ring), func(i int) bool { return r.ring[i] >= h })
	if idx == len(r.ring) {
		idx = 0
	}
	return r.targets[r.ring[idx]]
}

func (r *HashRing) RemoveByName(name string) bool {
	removed := false
	newRing := make([]uint32, 0)

	for _, h := range r.ring {
		if r.targets[h].Name == name {
			delete(r.targets, h)
			removed = true
		} else {
			newRing = append(newRing, h)
		}
	}

	if removed {
		r.ring = newRing
		sort.Slice(r.ring, func(i, j int) bool { return r.ring[i] < r.ring[j] })
	}

	return removed
}

// Size returns the number of UNIQUE nodes in the ring, not the numbers of total nodes (which is = nUniqueNodes * Replicas)
func (r *HashRing) Size() int {

	// Maybe simply doing "len(r.targets) / replicas could" be an easy solution? This one is more robust for sure.

	seen := make(map[*middleware.ProxyTarget]struct{})
	for _, t := range r.targets {
		seen[t] = struct{}{}
	}
	return len(seen)
}

func hash(s string) uint32 {
	h := fnv.New32a()
	_, err := h.Write([]byte(s))
	if err != nil {
		log.Printf("error hashing %s: %v", s, err)
	}
	return h.Sum32()
}
