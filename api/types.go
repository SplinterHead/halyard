package api

import "time"

// Service represents a Docker Swarm service summary
type Service struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Stack     string    `json:"stack"`
	Image     string    `json:"image"`
	Mode      string    `json:"mode"`
	Replicas  uint64    `json:"replicas"`
	Running   uint64    `json:"running"`
	Ports     []string  `json:"ports"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NodeStats represents system statistics from a Swarm node
type NodeStats struct {
	NodeID       string    `json:"node_id"`
	Hostname     string    `json:"hostname"`
	Status       string    `json:"status"`
	Availability string    `json:"availability"`
	Role         string    `json:"role"`
	Version      string    `json:"version"`
	IP           string    `json:"ip"`
	CPUUsage     float64   `json:"cpu_usage"`
	MemoryUsage  uint64    `json:"memory_usage"`
	MemoryTotal  uint64    `json:"memory_total"`
	DiskUsage    uint64    `json:"disk_usage"`
	Uptime       uint64    `json:"uptime"`
	Timestamp    time.Time `json:"timestamp"`
}

type NodeDetail struct {
	NodeStats
	OS            string            `json:"os"`
	Architecture  string            `json:"architecture"`
	CPUs          int               `json:"cpus"`
	Memory        int64             `json:"memory"`
	Labels        map[string]string `json:"labels"`
	EngineVersion string            `json:"engine_version"`
	Plugins       []string          `json:"plugins"`
}

// Stack represents a Docker Swarm stack (grouped by namespace label)
type Stack struct {
	Name            string    `json:"name"`
	Services        int       `json:"services"`
	TotalReplicas   uint64    `json:"total_replicas"`
	RunningReplicas uint64    `json:"running_replicas"`
	Status          string    `json:"status"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// VolumeInfo represents a Docker volume's metadata
type VolumeInfo struct {
	Name       string            `json:"name"`
	Node       string            `json:"node"`
	Stack      string            `json:"stack"`
	Driver     string            `json:"driver"`
	Type       string            `json:"type"`
	Mountpoint string            `json:"mountpoint"`
	CreatedAt  time.Time         `json:"created_at"`
	InUse      bool              `json:"in_use"`
	Labels     map[string]string `json:"labels"`
}

// NetworkInfo represents a Docker network's metadata
type NetworkInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Node      string    `json:"node"`
	Driver    string    `json:"driver"`
	Scope     string    `json:"scope"`
	Subnet    string    `json:"subnet"`
	Gateway   string    `json:"gateway"`
	Stack     string    `json:"stack"`
	CreatedAt time.Time `json:"created_at"`
}

type NetworkDetail struct {
	NetworkInfo
	Containers []NetworkContainer `json:"containers"`
	IPAMConfig []IPAMConfig       `json:"ipam_configs"`
	Options    map[string]string  `json:"options"`
	Labels     map[string]string  `json:"labels"`
	Internal   bool               `json:"internal"`
	Attachable bool               `json:"attachable"`
	Ingress    bool               `json:"ingress"`
}

type NetworkContainer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	IPv4 string `json:"ipv4"`
	IPv6 string `json:"ipv6"`
}

// ContainerInfo represents a Docker container's metadata
type ContainerInfo struct {
	ID        string    `json:"id"`
	Names     []string  `json:"names"`
	Image     string    `json:"image"`
	ImageID   string    `json:"image_id"`
	State     string    `json:"state"`
	Status    string    `json:"status"`
	Node      string    `json:"node"`
	Service   string    `json:"service"`
	Stack     string    `json:"stack"`
	UpToDate  bool      `json:"up_to_date"`
	CreatedAt time.Time `json:"created_at"`
}

type ContainerDetail struct {
	ContainerInfo
	Env     []string          `json:"env"`
	Labels  map[string]string `json:"labels"`
	Mounts  []Mount           `json:"mounts"`
	Network []string          `json:"networks"`
	Ports   []Port            `json:"ports"`
}

type Mount struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Type        string `json:"type"`
	RW          bool   `json:"rw"`
}

type Port struct {
	IP          string `json:"ip"`
	PrivatePort uint16 `json:"private_port"`
	PublicPort  uint16 `json:"public_port"`
	Type        string `json:"type"`
}

type FileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
}

// GitRepository represents a git repository source for Halyard
type GitRepository struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Username    string    `json:"username"`
	Token       string    `json:"token"`
	Description string    `json:"description"`
	LastStatus  string    `json:"last_status"`
	LastError   string    `json:"last_error"`
	CreatedAt   time.Time `json:"created_at"`
}

// GitSync represents a scheduled synchronization of a compose file from Git
type GitSync struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	RepositoryID        string     `json:"repository_id"`
	StackName           string     `json:"stack_name"`
	Branch              string     `json:"branch"`
	Path                string     `json:"path"`
	PullAdditionalFiles bool       `json:"pull_additional_files"`
	AutoSync            bool       `json:"auto_sync"`
	LastAppliedSHA      string     `json:"last_applied_sha"`
	LastSyncAt          *time.Time `json:"last_sync_at"`
	LastStatus          string     `json:"last_status"`
	LastError           string     `json:"last_error"`
	CreatedAt           *time.Time `json:"created_at"`
}

// Settings represents application-wide configuration
type Settings struct {
	GitSyncInterval    int `json:"git_sync_interval"` // In minutes
	GitSyncConcurrency int `json:"git_sync_concurrency"`
}

type DeploymentEvent struct {
	ID        string    `json:"id"`
	SyncID    string    `json:"sync_id"`
	StackName string    `json:"stack_name"` // Joined from syncs
	SHA       string    `json:"sha"`
	Status    string    `json:"status"`
	Logs      string    `json:"logs"`
	Timestamp time.Time `json:"timestamp"`
}

type SecretInfo struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	UpdatedAt time.Time         `json:"updated_at"`
	CreatedAt time.Time         `json:"created_at"`
	Labels    map[string]string `json:"labels"`
}

type ConfigInfo struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	UpdatedAt time.Time         `json:"updated_at"`
	CreatedAt time.Time         `json:"created_at"`
	Labels    map[string]string `json:"labels"`
}

type TaskInfo struct {
	ID           string    `json:"id"`
	ServiceID    string    `json:"service_id"`
	ServiceName  string    `json:"service_name"`
	NodeID       string    `json:"node_id"`
	NodeName     string    `json:"node_name"`
	State        string    `json:"state"`
	DesiredState string    `json:"desired_state"`
	Error        string    `json:"error"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type StackDetail struct {
	Stack
	Services   []Service       `json:"services_list"`
	Containers []ContainerInfo `json:"containers_list"`
	Volumes    []VolumeDetail  `json:"volumes_list"`
	Tasks      []TaskInfo      `json:"tasks_list"`
}

type VolumeDetail struct {
	Name     string `json:"name"`
	External bool   `json:"external"`
}

type ServiceDetail struct {
	Service
	Labels        map[string]string `json:"labels"`
	RestartPolicy string            `json:"restart_policy"`
	Env           []string          `json:"env"`
	Constraints   []string          `json:"constraints"`
}

// Registry represents a Docker registry with credentials
type Registry struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"` // Human friendly name
	URL       string    `json:"url"`  // Registry URL (e.g. index.docker.io/v1/ or custom)
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
}

// NetworkCreateRequest represents the data needed to create a new Docker network
type NetworkCreateRequest struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Attachable bool              `json:"attachable"`
	Internal   bool              `json:"internal"`
	IPAM       *IPAMConfig       `json:"ipam,omitempty"`
	Options    map[string]string `json:"options,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

type IPAMConfig struct {
	Subnet  string `json:"subnet,omitempty"`
	Gateway string `json:"gateway,omitempty"`
}

// ImageInfo represents metadata of a Docker image stored on a Swarm node
type ImageInfo struct {
	ID           string    `json:"id"`
	Repository   string    `json:"repository"`
	Tag          string    `json:"tag"`
	Node         string    `json:"node"`
	NodeID       string    `json:"node_id"`
	Size         int64     `json:"size"`
	Architecture string    `json:"architecture"`
	InUse        bool      `json:"in_use"`
	CreatedAt    time.Time `json:"created_at"`
}

// PruneRequest represents a requested Docker prune operation with customizable filters
type PruneRequest struct {
	Containers bool `json:"containers"`
	Networks   bool `json:"networks"`
	Volumes    bool `json:"volumes"`
	Images     bool `json:"images"`
	ImagesAll  bool `json:"images_all"`
}

// DockerHubRateLimit represents Docker Hub pull rate limit headers
type DockerHubRateLimit struct {
	Limit     int `json:"limit"`
	Remaining int `json:"remaining"`
	Reset     int `json:"reset"`
}

// User represents the system user
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	RealName  string    `json:"real_name"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterRequest represents the payload to register the first user
type RegisterRequest struct {
	Username string `json:"username"`
	RealName string `json:"real_name"`
	Password string `json:"password"`
}

// LoginRequest represents the payload to authenticate
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse represents the login/onboarding success payload
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

