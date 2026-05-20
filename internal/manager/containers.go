package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
)

type ContainerAggregator struct {
	docker *docker.Client
	client *http.Client
}

func NewContainerAggregator(cli *docker.Client) *ContainerAggregator {
	return &ContainerAggregator{
		docker: cli,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (m *ContainerAggregator) ListAllContainers(ctx context.Context) ([]api.ContainerInfo, error) {
	// Get nodes to know hostnames
	nodes, err := m.docker.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return nil, err
	}
	nodeMap := make(map[string]string)
	for _, n := range nodes {
		nodeMap[n.ID] = n.Description.Hostname
	}

	// Get agent tasks to find their IPs
	agentTasks, err := m.docker.TaskList(ctx, types.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("service", "halyard_agent"), filters.Arg("desired-state", "running")),
	})
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	allContainers := make([]api.ContainerInfo, 0)

	for _, task := range agentTasks {
		if task.Status.State == "running" && len(task.NetworksAttachments) > 0 {
			var ip string
			for _, net := range task.NetworksAttachments {
				if len(net.Addresses) > 0 {
					fullIP := net.Addresses[0]
					if i := strings.Index(fullIP, "/"); i != -1 {
						ip = fullIP[:i]
						break
					}
				}
			}

			if ip != "" {
				wg.Add(1)
				go func(taskIP, nodeID string) {
					defer wg.Done()
					resp, err := m.client.Get(fmt.Sprintf("http://%s:9090/containers", taskIP))
					if err != nil {
						return
					}
					defer resp.Body.Close()

					var conts []api.ContainerInfo
					if err := json.NewDecoder(resp.Body).Decode(&conts); err == nil {
						mu.Lock()
						for i := range conts {
							conts[i].Node = nodeMap[nodeID]
							allContainers = append(allContainers, conts[i])
						}
						mu.Unlock()
					}
				}(ip, task.NodeID)
			}
		}
	}

	wg.Wait()
	return allContainers, nil
}
func (m *ContainerAggregator) GetContainerDetail(ctx context.Context, id, nodeName string) (api.ContainerDetail, error) {
	agentIP, err := m.getAgentIP(ctx, nodeName)
	if err != nil {
		return api.ContainerDetail{}, err
	}

	// 2. Fetch from agent
	resp, err := m.client.Get(fmt.Sprintf("http://%s:9090/containers/detail?id=%s", agentIP, id))
	if err != nil {
		return api.ContainerDetail{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return api.ContainerDetail{}, fmt.Errorf("agent returned status %d", resp.StatusCode)
	}

	var detail api.ContainerDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return api.ContainerDetail{}, err
	}

	detail.Node = nodeName
	return detail, nil
}

func (m *ContainerAggregator) ProxyLogs(ctx context.Context, id, nodeName string, w http.ResponseWriter) error {
	agentIP, err := m.getAgentIP(ctx, nodeName)
	if err != nil {
		return err
	}

	resp, err := m.client.Get(fmt.Sprintf("http://%s:9090/containers/logs?id=%s", agentIP, id))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("agent returned status %d", resp.StatusCode)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Transfer-Encoding", "chunked")
	
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, err := resp.Body.Read(buf)
			if n > 0 {
				w.Write(buf[:n])
				flusher.Flush()
			}
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
		}
	}
}

func (m *ContainerAggregator) getAgentIP(ctx context.Context, nodeName string) (string, error) {
	agentTasks, err := m.docker.TaskList(ctx, types.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("service", "halyard_agent"), filters.Arg("desired-state", "running")),
	})
	if err != nil {
		return "", err
	}

	nodes, err := m.docker.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return "", err
	}
	var nodeID string
	for _, n := range nodes {
		if n.Description.Hostname == nodeName {
			nodeID = n.ID
			break
		}
	}

	if nodeID == "" {
		return "", fmt.Errorf("node %s not found", nodeName)
	}

	for _, task := range agentTasks {
		if task.NodeID == nodeID && len(task.NetworksAttachments) > 0 {
			for _, net := range task.NetworksAttachments {
				if len(net.Addresses) > 0 {
					fullIP := net.Addresses[0]
					if i := strings.Index(fullIP, "/"); i != -1 {
						return fullIP[:i], nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("agent not found on node %s", nodeName)
}
func (m *ContainerAggregator) StreamLogsWS(ctx context.Context, id, nodeName string, w http.ResponseWriter, r *http.Request) error {
	agentIP, err := m.getAgentIP(ctx, nodeName)
	if err != nil {
		return err
	}

	// 1. Upgrade client connection
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	defer clientConn.Close()

	// 2. Connect to agent WS
	agentURL := fmt.Sprintf("ws://%s:9090/containers/logs?id=%s", agentIP, id)
	agentConn, _, err := websocket.DefaultDialer.Dial(agentURL, nil)
	if err != nil {
		clientConn.WriteMessage(websocket.TextMessage, []byte("Error connecting to agent: "+err.Error()))
		return err
	}
	defer agentConn.Close()

	// 3. Proxy loop
	errChan := make(chan error, 2)
	
	// Agent -> Client
	go func() {
		for {
			mt, message, err := agentConn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			if err := clientConn.WriteMessage(mt, message); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Client -> Agent (for close messages etc)
	go func() {
		for {
			mt, message, err := clientConn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			if err := agentConn.WriteMessage(mt, message); err != nil {
				errChan <- err
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

