package kube

// PodInfo holds the Kubernetes metadata for a pod that is relevant to
// container diagnosis.
type PodInfo struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Node        string            `json:"node"`
	Phase       string            `json:"phase"`
	Reason      string            `json:"reason,omitempty"`
	Message     string            `json:"message,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Containers  []PodContainer    `json:"containers"`
	QoS         string            `json:"qos,omitempty"`
	Restarts    int               `json:"restarts"`
}

// PodContainer is one container within a pod.
type PodContainer struct {
	Name         string `json:"name"`
	Image        string `json:"image"`
	ContainerID  string `json:"container_id"`
	Ready        bool   `json:"ready"`
	RestartCount int    `json:"restart_count"`
	State        string `json:"state"`
	LastExitCode int    `json:"last_exit_code,omitempty"`
	LastReason   string `json:"last_reason,omitempty"`
	LastOOM      bool   `json:"last_oom,omitempty"`
}

// Client abstracts Kubernetes CLI access for testing.
type Client interface {
	GetPod(namespace, name string) (PodInfo, error)
	ResolveContainer(podName, containerName, namespace string) (containerID string, err error)
	DetectNamespace() string
}
