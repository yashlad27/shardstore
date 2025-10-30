package hash

import (
	"crypto/md5"
	"fmt"
	"sort"
	"sync"
)

const (
	// DefaultVirtualNodes is the default number of virtual nodes per physical node
	DefaultVirtualNodes = 150
)

// ConsistentHash implements consistent hashing with virtual nodes
type ConsistentHash struct {
	mu            sync.RWMutex
	hashRing      []uint32
	nodes         map[uint32]string
	virtualNodes  int
	replicaFactor int
}

// NewConsistentHash creates a new consistent hash ring
func NewConsistentHash(virtualNodes, replicaFactor int) *ConsistentHash {
	if virtualNodes <= 0 {
		virtualNodes = DefaultVirtualNodes
	}
	if replicaFactor <= 0 {
		replicaFactor = 2
	}
	return &ConsistentHash{
		hashRing:      []uint32{},
		nodes:         make(map[uint32]string),
		virtualNodes:  virtualNodes,
		replicaFactor: replicaFactor,
	}
}

// AddNode adds a physical node to the hash ring
func (ch *ConsistentHash) AddNode(nodeID string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// Add virtual nodes
	for i := 0; i < ch.virtualNodes; i++ {
		virtualKey := fmt.Sprintf("%s#%d", nodeID, i)
		hash := ch.hashKey(virtualKey)
		ch.hashRing = append(ch.hashRing, hash)
		ch.nodes[hash] = nodeID
	}

	sort.Slice(ch.hashRing, func(i, j int) bool {
		return ch.hashRing[i] < ch.hashRing[j]
	})
}

// RemoveNode removes a physical node from the hash ring
func (ch *ConsistentHash) RemoveNode(nodeID string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// Remove virtual nodes
	for i := 0; i < ch.virtualNodes; i++ {
		virtualKey := fmt.Sprintf("%s#%d", nodeID, i)
		hash := ch.hashKey(virtualKey)
		
		// Find and remove from ring
		idx := sort.Search(len(ch.hashRing), func(i int) bool {
			return ch.hashRing[i] >= hash
		})
		if idx < len(ch.hashRing) && ch.hashRing[idx] == hash {
			ch.hashRing = append(ch.hashRing[:idx], ch.hashRing[idx+1:]...)
		}
		delete(ch.nodes, hash)
	}
}

// GetNodes returns the primary and replica nodes for a given key
func (ch *ConsistentHash) GetNodes(key string) []string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if len(ch.hashRing) == 0 {
		return nil
	}

	hash := ch.hashKey(key)
	idx := ch.search(hash)

	// Get unique nodes for replication
	nodeSet := make(map[string]bool)
	nodes := []string{}
	
	for i := 0; len(nodes) < ch.replicaFactor && i < len(ch.hashRing); i++ {
		ringIdx := (idx + i) % len(ch.hashRing)
		nodeID := ch.nodes[ch.hashRing[ringIdx]]
		
		if !nodeSet[nodeID] {
			nodeSet[nodeID] = true
			nodes = append(nodes, nodeID)
		}
	}

	return nodes
}

// GetPrimaryNode returns the primary node for a given key
func (ch *ConsistentHash) GetPrimaryNode(key string) string {
	nodes := ch.GetNodes(key)
	if len(nodes) > 0 {
		return nodes[0]
	}
	return ""
}

// GetAllNodes returns all physical nodes in the ring
func (ch *ConsistentHash) GetAllNodes() []string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	nodeSet := make(map[string]bool)
	for _, nodeID := range ch.nodes {
		nodeSet[nodeID] = true
	}

	nodes := make([]string, 0, len(nodeSet))
	for nodeID := range nodeSet {
		nodes = append(nodes, nodeID)
	}
	return nodes
}

// hashKey generates a hash for a given key
func (ch *ConsistentHash) hashKey(key string) uint32 {
	hash := md5.Sum([]byte(key))
	return uint32(hash[0])<<24 | uint32(hash[1])<<16 | uint32(hash[2])<<8 | uint32(hash[3])
}

// search finds the index of the first node >= hash
func (ch *ConsistentHash) search(hash uint32) int {
	idx := sort.Search(len(ch.hashRing), func(i int) bool {
		return ch.hashRing[i] >= hash
	})
	
	if idx >= len(ch.hashRing) {
		idx = 0
	}
	return idx
}
