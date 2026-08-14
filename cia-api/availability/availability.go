package availability

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type NodeStatus struct {
	ID      string `json:"id"`
	Active  bool   `json:"active"`
	Healthy bool   `json:"healthy"`
}

type Cluster struct {
	mu       sync.RWMutex
	nodes    []NodeStatus
	failRate float64 // probability a node appears down (0.0-1.0)
}

func NewCluster(nodeCount int, failRate float64) *Cluster {
	nodes := make([]NodeStatus, nodeCount)
	for i := 0; i < nodeCount; i++ {
		nodes[i] = NodeStatus{
			ID:      "node-" + string(rune('A'+i)),
			Active:  true,
			Healthy: true,
		}
	}
	c := &Cluster{nodes: nodes, failRate: failRate}
	go c.healthLoop()
	return c
}

func (c *Cluster) healthLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		for i := range c.nodes {
			c.nodes[i].Healthy = rand.Float64() > c.failRate
		}
		c.mu.Unlock()
	}
}

func (c *Cluster) Status() []NodeStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]NodeStatus, len(c.nodes))
	copy(out, c.nodes)
	return out
}

func (c *Cluster) ActiveNode() (int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for i, n := range c.nodes {
		if n.Active && n.Healthy {
			return i, true
		}
	}
	return -1, false
}

func (c *Cluster) HandleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes": c.Status(),
	})
}

func (c *Cluster) HandleRequest(w http.ResponseWriter, r *http.Request) {
	idx, ok := c.ActiveNode()
	if !ok {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "all nodes down"})
		return
	}
	c.mu.RLock()
	node := c.nodes[idx]
	c.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"node":   node.ID,
		"status": "ok",
	})
}

// SetFailRate updates the probability of a node being down
func (c *Cluster) SetFailRate(rate float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failRate = rate
}
