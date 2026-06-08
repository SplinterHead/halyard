package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/docker/docker/api/types"
	"github.com/SplinterHead/halyard/api"
	"github.com/SplinterHead/halyard/internal/pkg/agentclient"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
)

type ContainerAggregator struct {
	docker   *docker.Client
	client   *http.Client
	agentDir *AgentDirectory
}

func NewContainerAggregator(cli *docker.Client, agentDir *AgentDirectory) *ContainerAggregator {
	return &ContainerAggregator{
		docker:   cli,
		client:   &http.Client{Timeout: 5 * time.Second},
		agentDir: agentDir,
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

	// Get active agents from directory
	activeAgents := m.agentDir.GetAllAgents()

	var wg sync.WaitGroup
	var mu sync.Mutex
	allContainers := make([]api.ContainerInfo, 0)

	for _, agent := range activeAgents {
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
		}(agent.IP, agent.NodeID)
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

	if err := agentclient.ParseError(resp, http.StatusOK); err != nil {
		return api.ContainerDetail{}, err
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

	if err := agentclient.ParseError(resp, http.StatusOK); err != nil {
		return err
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
	ip, ok := m.agentDir.GetAgentIPByHostname(nodeName)
	if !ok {
		return "", fmt.Errorf("agent not found on node %s in AgentDirectory", nodeName)
	}
	return ip, nil
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

func (m *ContainerAggregator) StartContainer(ctx context.Context, id, nodeName string) error {
	agentIP, err := m.getAgentIP(ctx, nodeName)
	if err != nil {
		return err
	}
	resp, err := m.client.Post(fmt.Sprintf("http://%s:9090/containers/start?id=%s", agentIP, id), "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return agentclient.ParseError(resp, http.StatusNoContent, http.StatusOK)
}

func (m *ContainerAggregator) StopContainer(ctx context.Context, id, nodeName string) error {
	agentIP, err := m.getAgentIP(ctx, nodeName)
	if err != nil {
		return err
	}
	resp, err := m.client.Post(fmt.Sprintf("http://%s:9090/containers/stop?id=%s", agentIP, id), "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return agentclient.ParseError(resp, http.StatusNoContent, http.StatusOK)
}

func (m *ContainerAggregator) RestartContainer(ctx context.Context, id, nodeName string) error {
	agentIP, err := m.getAgentIP(ctx, nodeName)
	if err != nil {
		return err
	}
	resp, err := m.client.Post(fmt.Sprintf("http://%s:9090/containers/restart?id=%s", agentIP, id), "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return agentclient.ParseError(resp, http.StatusNoContent, http.StatusOK)
}

func (m *ContainerAggregator) DeleteContainer(ctx context.Context, id, nodeName string, force bool) error {
	agentIP, err := m.getAgentIP(ctx, nodeName)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("http://%s:9090/containers/remove?id=%s&force=%t", agentIP, id, force), nil)
	if err != nil {
		return err
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return agentclient.ParseError(resp, http.StatusNoContent, http.StatusOK)
}

func (m *ContainerAggregator) ProxyExecWS(ctx context.Context, id, nodeName, shell string, w http.ResponseWriter, r *http.Request) error {
	agentIP, err := m.getAgentIP(ctx, nodeName)
	if err != nil {
		return err
	}

	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	defer clientConn.Close()

	agentURL := fmt.Sprintf("ws://%s:9090/containers/exec?id=%s&shell=%s", agentIP, id, shell)
	agentConn, _, err := websocket.DefaultDialer.Dial(agentURL, nil)
	if err != nil {
		clientConn.WriteMessage(websocket.TextMessage, []byte("\r\nError connecting to agent: "+err.Error()+"\r\n"))
		return err
	}
	defer agentConn.Close()

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

	// Client -> Agent
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

