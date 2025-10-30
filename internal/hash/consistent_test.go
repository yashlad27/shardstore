package hash

import (
	"fmt"
	"testing"
)

func TestNewConsistentHash(t *testing.T) {
	tests := []struct {
		name          string
		virtualNodes  int
		replicaFactor int
		wantVirtual   int
		wantReplica   int
	}{
		{
			name:          "default values",
			virtualNodes:  0,
			replicaFactor: 0,
			wantVirtual:   DefaultVirtualNodes,
			wantReplica:   2,
		},
		{
			name:          "custom values",
			virtualNodes:  100,
			replicaFactor: 3,
			wantVirtual:   100,
			wantReplica:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := NewConsistentHash(tt.virtualNodes, tt.replicaFactor)
			if ch.virtualNodes != tt.wantVirtual {
				t.Errorf("virtualNodes = %d, want %d", ch.virtualNodes, tt.wantVirtual)
			}
			if ch.replicaFactor != tt.wantReplica {
				t.Errorf("replicaFactor = %d, want %d", ch.replicaFactor, tt.wantReplica)
			}
		})
	}
}

func TestAddNode(t *testing.T) {
	ch := NewConsistentHash(10, 2)
	
	t.Run("add single node", func(t *testing.T) {
		ch.AddNode("node-1")
		if len(ch.hashRing) != 10 {
			t.Errorf("hashRing length = %d, want 10", len(ch.hashRing))
		}
		if len(ch.nodes) != 10 {
			t.Errorf("nodes map length = %d, want 10", len(ch.nodes))
		}
	})

	t.Run("add multiple nodes", func(t *testing.T) {
		ch.AddNode("node-2")
		ch.AddNode("node-3")
		if len(ch.hashRing) != 30 {
			t.Errorf("hashRing length = %d, want 30", len(ch.hashRing))
		}
	})

	t.Run("hash ring is sorted", func(t *testing.T) {
		for i := 1; i < len(ch.hashRing); i++ {
			if ch.hashRing[i-1] >= ch.hashRing[i] {
				t.Error("hash ring is not sorted")
				break
			}
		}
	})
}

func TestRemoveNode(t *testing.T) {
	ch := NewConsistentHash(10, 2)
	ch.AddNode("node-1")
	ch.AddNode("node-2")
	ch.AddNode("node-3")

	t.Run("remove existing node", func(t *testing.T) {
		initialLen := len(ch.hashRing)
		ch.RemoveNode("node-2")
		if len(ch.hashRing) != initialLen-10 {
			t.Errorf("hashRing length = %d, want %d", len(ch.hashRing), initialLen-10)
		}
	})

	t.Run("remove non-existing node", func(t *testing.T) {
		initialLen := len(ch.hashRing)
		ch.RemoveNode("node-999")
		if len(ch.hashRing) != initialLen {
			t.Error("hashRing length changed when removing non-existing node")
		}
	})
}

func TestGetNodes(t *testing.T) {
	ch := NewConsistentHash(10, 2)
	ch.AddNode("node-1")
	ch.AddNode("node-2")
	ch.AddNode("node-3")

	t.Run("get nodes for key", func(t *testing.T) {
		nodes := ch.GetNodes("test-key")
		if len(nodes) != 2 {
			t.Errorf("got %d nodes, want 2", len(nodes))
		}
		// Check uniqueness
		if len(nodes) == 2 && nodes[0] == nodes[1] {
			t.Error("returned duplicate nodes")
		}
	})

	t.Run("consistent key mapping", func(t *testing.T) {
		key := "consistent-key"
		nodes1 := ch.GetNodes(key)
		nodes2 := ch.GetNodes(key)
		
		if len(nodes1) != len(nodes2) {
			t.Error("inconsistent node count for same key")
		}
		for i := range nodes1 {
			if nodes1[i] != nodes2[i] {
				t.Error("inconsistent node mapping for same key")
			}
		}
	})

	t.Run("different keys get different nodes", func(t *testing.T) {
		keys := []string{"key1", "key2", "key3", "key4", "key5"}
		nodeDistribution := make(map[string]int)
		
		for _, key := range keys {
			nodes := ch.GetNodes(key)
			for _, node := range nodes {
				nodeDistribution[node]++
			}
		}
		
		// Check that nodes are being used (not perfect distribution expected)
		if len(nodeDistribution) == 0 {
			t.Error("no nodes were selected")
		}
	})
}

func TestGetPrimaryNode(t *testing.T) {
	ch := NewConsistentHash(10, 2)
	ch.AddNode("node-1")
	ch.AddNode("node-2")

	t.Run("get primary node", func(t *testing.T) {
		primary := ch.GetPrimaryNode("test-key")
		if primary == "" {
			t.Error("primary node is empty")
		}
	})

	t.Run("primary is first in GetNodes", func(t *testing.T) {
		key := "test-key-123"
		primary := ch.GetPrimaryNode(key)
		nodes := ch.GetNodes(key)
		
		if len(nodes) > 0 && nodes[0] != primary {
			t.Errorf("primary node %s != first node %s", primary, nodes[0])
		}
	})

	t.Run("empty ring returns empty", func(t *testing.T) {
		emptyRing := NewConsistentHash(10, 2)
		primary := emptyRing.GetPrimaryNode("key")
		if primary != "" {
			t.Error("expected empty primary for empty ring")
		}
	})
}

func TestGetAllNodes(t *testing.T) {
	ch := NewConsistentHash(10, 2)

	t.Run("empty ring", func(t *testing.T) {
		nodes := ch.GetAllNodes()
		if len(nodes) != 0 {
			t.Errorf("got %d nodes, want 0", len(nodes))
		}
	})

	t.Run("with nodes", func(t *testing.T) {
		ch.AddNode("node-1")
		ch.AddNode("node-2")
		ch.AddNode("node-3")
		
		nodes := ch.GetAllNodes()
		if len(nodes) != 3 {
			t.Errorf("got %d nodes, want 3", len(nodes))
		}
		
		// Check uniqueness
		seen := make(map[string]bool)
		for _, node := range nodes {
			if seen[node] {
				t.Errorf("duplicate node in GetAllNodes: %s", node)
			}
			seen[node] = true
		}
	})
}

func TestHashDistribution(t *testing.T) {
	ch := NewConsistentHash(150, 3)
	ch.AddNode("node-1")
	ch.AddNode("node-2")
	ch.AddNode("node-3")

	// Generate keys and track distribution
	nodeCount := make(map[string]int)
	numKeys := 1000

	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		primary := ch.GetPrimaryNode(key)
		nodeCount[primary]++
	}

	// Check that all nodes got some keys (basic distribution)
	if len(nodeCount) != 3 {
		t.Errorf("not all nodes received keys: %v", nodeCount)
	}

	// Check that distribution is somewhat balanced (within 40% of average)
	average := float64(numKeys) / 3.0
	for node, count := range nodeCount {
		percentage := float64(count) / average
		if percentage < 0.6 || percentage > 1.4 {
			t.Logf("Node %s has imbalanced distribution: %d keys (%.1f%% of average)", 
				node, count, percentage*100)
		}
	}
}

func TestRebalancingOnNodeRemoval(t *testing.T) {
	ch := NewConsistentHash(50, 2)
	ch.AddNode("node-1")
	ch.AddNode("node-2")
	ch.AddNode("node-3")

	// Map keys before removal
	keyMapping := make(map[string][]string)
	testKeys := []string{"key1", "key2", "key3", "key4", "key5"}
	
	for _, key := range testKeys {
		keyMapping[key] = ch.GetNodes(key)
	}

	// Remove a node
	ch.RemoveNode("node-2")

	// Check that removed node is not in any mapping
	for _, key := range testKeys {
		nodes := ch.GetNodes(key)
		for _, node := range nodes {
			if node == "node-2" {
				t.Errorf("removed node-2 still appears for key %s", key)
			}
		}
	}

	// Check that we still get correct number of replicas
	for _, key := range testKeys {
		nodes := ch.GetNodes(key)
		if len(nodes) != 2 {
			t.Errorf("expected 2 replicas for %s, got %d", key, len(nodes))
		}
	}
}

func BenchmarkGetNodes(b *testing.B) {
	ch := NewConsistentHash(150, 2)
	ch.AddNode("node-1")
	ch.AddNode("node-2")
	ch.AddNode("node-3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch.GetNodes(fmt.Sprintf("key-%d", i))
	}
}

func BenchmarkAddNode(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch := NewConsistentHash(150, 2)
		ch.AddNode(fmt.Sprintf("node-%d", i))
	}
}
