package manager

import (
	"sync"
	"time"
)

type AgentRecord struct {
	NodeID   string    `json:"node_id"`
	IP       string    `json:"ip"`
	Hostname string    `json:"hostname"`
	LastSeen time.Time `json:"last_seen"`
}

type AgentDirectory struct {
	mu            sync.RWMutex
	recordsByNode map[string]AgentRecord
	recordsByName map[string]AgentRecord
}

func NewAgentDirectory() *AgentDirectory {
	return &AgentDirectory{
		recordsByNode: make(map[string]AgentRecord),
		recordsByName: make(map[string]AgentRecord),
	}
}

func (d *AgentDirectory) Update(nodeID, ip, hostname string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	record := AgentRecord{
		NodeID:   nodeID,
		IP:       ip,
		Hostname: hostname,
		LastSeen: time.Now(),
	}

	d.recordsByNode[nodeID] = record
	if hostname != "" {
		d.recordsByName[hostname] = record
	}
}

func (d *AgentDirectory) GetAgentIP(nodeID string) (string, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	record, ok := d.recordsByNode[nodeID]
	if !ok {
		return "", false
	}
	return record.IP, true
}

func (d *AgentDirectory) GetAgentIPByHostname(hostname string) (string, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	record, ok := d.recordsByName[hostname]
	if !ok {
		// Fallback to checking records by Node ID if the ID itself was passed
		record, ok = d.recordsByNode[hostname]
		if !ok {
			return "", false
		}
	}
	return record.IP, true
}

func (d *AgentDirectory) GetAllAgents() []AgentRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()

	agents := make([]AgentRecord, 0, len(d.recordsByNode))
	for _, record := range d.recordsByNode {
		// Only return agents seen in the last 1 minute
		if time.Since(record.LastSeen) < 1*time.Minute {
			agents = append(agents, record)
		}
	}
	return agents
}
