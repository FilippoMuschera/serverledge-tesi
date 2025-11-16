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
	ring     []uint32                           // actual ring with hash of nodes
	targets  map[uint32]*middleware.ProxyTarget // mapping hash(es) <-> node. Each node will have #replicas entries in the ring
}

func NewHashRing(replicas int) *HashRing {
	return &HashRing{
		replicas: replicas,
		ring:     make([]uint32, 0),
		targets:  make(map[uint32]*middleware.ProxyTarget),
	}
}

func (r *HashRing) Add(t *middleware.ProxyTarget) {
	// put replicas in the ring. To do so we'll hash the node's name + an incrementing number
	for i := 0; i < r.replicas; i++ {
		key := fmt.Sprintf("%s#%d", t.Name, i)
		h := hash(key)
		r.ring = append(r.ring, h)
		r.targets[h] = t
	}
	sort.Slice(r.ring, func(i, j int) bool { return r.ring[i] < r.ring[j] }) // sort the ring by hash
}

func (r *HashRing) Get(key string) *middleware.ProxyTarget {
	if len(r.ring) == 0 {
		return nil
	}

	h := hash(key)
	// we'll return the node whose hash is the next in the ring, starting from the hash of the function's name
	idx := sort.Search(len(r.ring), func(i int) bool { return r.ring[i] >= h })
	if idx == len(r.ring) {
		idx = 0
	}
	return r.targets[r.ring[idx]] // here we use the map to get the node corresponding to the hash
}

func (r *HashRing) RemoveByName(name string) bool {
	removed := false
	newRing := make([]uint32, 0)

	// We'll delete all entries for this node from the targets' map, and generate a new ring without them.

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

// hash function uses the FNV-1a function. It has good distribution and is fast to compute. It's not cryptographically safe,
// but should be good enough for our purposes (consistent-hashing).
func hash(s string) uint32 {
	h := fnv.New32a()
	_, err := h.Write([]byte(s))
	if err != nil {
		log.Printf("error hashing %s: %v", s, err)
	}
	return h.Sum32()
}
