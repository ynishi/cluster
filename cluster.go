package cluster

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"time"
	//"github.com/docker/docker/client"
)

type ClusterService interface {
	// container runtime version
	Version() (Version, error)
	// image of container
	Image() (Image, error)
	// default options for container
	Options() (ContainerOptions, error)
	// get containers in cluster
	Containers(all bool) (Containers, error)
	// get container status by uid or (name and nodeName).
	ContainerStatus(uid UID, name string, nodeName string) (*ContainerStatus, error)
	// create new container
	CrateContainer() error
	// run container
	RunContainer(container *Container) error
	// kill running container
	KillContainer(runningContainer *Container) error
	// get nodes in cluster
	Nodes(all bool) ([]*Node, error)
	// create new node
	CrateNode() error
	// run node
	RunNode(node *Node) error
	// kill running node(if it is vm, shutdown) wait for gracePeriod(ms).
	// if it is over, try to force kill.
	KillNode(runningNode Node, gracePeriod int) error
	// get cluster status
	Status() (ClusterStatus, error)
	// get node status by uid or name.
	NodeStatus(uid UID, name string) (NodeStatus, error)
	// flush node status
	FlushNodes() error
	// flush container status
	FlushContainers() error
}

type UID string
type Version string

type DefaultClusterService struct {
	ClusterService
	version           Version
	image             *Image
	options           ContainerOptions
	containers        Containers
	containerStatuses ContainerStatuses
	nodes             Nodes
	nodeStatuses      NodeStatuses
	nodesById         map[UID]*Node
	nodesByName       map[string]*Node
	maxNameI          int
}

func NewDefaultClusterService(version Version, image *Image) *DefaultClusterService {
	return &DefaultClusterService{
		version:    version,
		image:      image,
		containers: Containers{},
		containerStatuses: ContainerStatuses{},
		nodes: Nodes{},
		nodeStatuses: NodeStatuses{},
		nodesById: make(map[UID]*Node),
		nodesByName: make(map[string]*Node),
		maxNameI:0,
	}
}

type ClusterStatus struct {
	ClusterState ClusterState
	Reason       string
}

type ClusterState string

func (dcs *DefaultClusterService) Version() (Version, error) {
	if dcs.version == "" {
		return "", errors.New("not set version")
	}
	return dcs.version, nil
}

func (dcs *DefaultClusterService) Image() (*Image, error) {
	if dcs.image == nil {
		return nil, errors.New("not set image")
	}
	return dcs.image, nil
}

func (dcs *DefaultClusterService) Options() (ContainerOptions, error) {
	if dcs.options == nil {
		return nil, errors.New("not set image")
	}
	return dcs.options, nil
}

func (dcs *DefaultClusterService) Containers(all bool) (Containers, error) {
	if all {
		return dcs.containers, nil
	}
	res := Containers{}
	for _, c := range dcs.containers {
		cs, err := dcs.ContainerStatus(c.Id, "", "")
		if err != nil {
			return nil, err
		}
		if cs.ContainerState != ContainerExited && cs.ContainerState != ContainerUnknown {
			res = append(res, c)
		}
	}
	return res, nil
}

func (dcs *DefaultClusterService) ContainerStatus(uid UID, name string, nodeName string) (*ContainerStatus, error) {
	if uid == "" && (name == "" || nodeName == "") {
		return nil, errors.New("uid or (name and nodeName) required")
	}

	for _, cs := range dcs.containerStatuses {
		if cs.Id == uid || (cs.Name == name && cs.NodeName == nodeName) {
			return cs, nil
		}
	}

	containerFound := false
	for _, c := range dcs.containers {
		if c.Id == uid || (c.Name == name && c.NodeName == nodeName) {
			containerFound = true
		}
	}
	if containerFound {
		return nil, fmt.Errorf("not found container for uid:%v, name:%v, nodeName:%v", uid, name, nodeName)
	}
	return nil, fmt.Errorf("not found container status for uid:%v, name:%v, nodeName:%v", uid, name, nodeName)
}

func (dcs *DefaultClusterService) CreateContainer() (*Container, error) {
	image, err := dcs.Image()
	if err != nil {
		return nil, err
	}
	node := dcs.minWorkingNode()
	if node == nil {
		return nil, errors.New("no valid node")
	}
	containerId := genUID()
	options, err := dcs.Options()
	if err != nil {
		return nil, err
	}
	container := NewContainer(containerId, "", "", node.Id, node.Name, image, "", options)
	dcs.containers = append(dcs.containers, container)
	dcs.containerStatuses = append(dcs.containerStatuses, container.ContainerStatus)
	return container, nil
}

func (dcs *DefaultClusterService) RunContainer(container *Container) error {
	if container.ContainerStatus.ContainerState == ContainerRunning {
		return fmt.Errorf("already running:%v", container.Name)
	}
	node := dcs.findNodeById(container.NodeId)
	err := node.RunContainer(container)
	return err
}

func (dcs *DefaultClusterService) CreateNode() (*Node, error) {
	nodeId := genUID()
	nodeName := dcs.genNodeName()
	node := &Node{
		Id:   nodeId,
		Name: nodeName,
	}
	dcs.nodes = append(dcs.nodes, node)
	dcs.nodesById[nodeId] = node
	dcs.nodesByName[nodeName] = node
	return node, nil
}

func (dcs *DefaultClusterService) minWorkingNode() *Node {
	nodes, _ := dcs.Nodes(false)
	return nodes[0]
}

func (dcs *DefaultClusterService) genNodeName() string {
	nodes, err := dcs.Nodes(true)
	if err != nil {
		return ""
	}
	if dcs.maxNameI >= 100000 {
		dcs.maxNameI = len(dcs.nodes)
	}
	var name string
	valid := false
	for i := dcs.maxNameI + 1; !valid || i < 100000; i++ {
		name := fmt.Sprintf("node-%d", i)
		for _, node := range nodes {
			if node.Name != name {
				valid = true
				dcs.maxNameI = i
				break
			}
		}
	}
	if valid {
		return name
	} else {
		return ""
	}
}

func (dcs *DefaultClusterService) findNodeById(id UID) *Node {
	return dcs.nodesById[id]
}

func (dcs *DefaultClusterService) findNodeByName(name string) *Node {
	return dcs.nodesByName[name]
}

type Image struct {
	// name of container image
	Name string
	// full name of container image, formatted: registory/name:tag
	FullName string
}

func NewImage(fullName string) (*Image, error) {
	index := strings.Index(fullName, ":")
	if index < 0 {
		return nil, errors.New("not found sep \":\"")
	}
	if strings.Index(fullName, ":") == 0 {
		return nil, errors.New("name not found")
	}
	splitList := strings.Split(fullName, ":")
	return &Image{
		Name:     splitList[0],
		FullName: fullName,
	}, nil
}

type Container struct {
	// uuid
	Id UID
	// container name on node
	Name string
	// container hash on node
	Hash string
	// uuid of Node running on
	NodeId UID
	// name of Node running on
	NodeName string
	// container status
	ContainerStatus *ContainerStatus
	// container image
	Image *Image
	// image id on node
	ImageId string
	// options for run
	ContainerOptions ContainerOptions
}

func NewContainer(id UID, name string, hash string, nodeId UID, nodeName string, image *Image, imageId string, options ContainerOptions) *Container {
	containerStatus := NewContainerStatus(id, name, nodeName)
	return &Container{
		Id:               id,
		Name:             name,
		Hash:             hash,
		NodeId:           nodeId,
		NodeName:         nodeName,
		ContainerStatus:  containerStatus,
		ContainerOptions: options,
		Image:            image,
		ImageId:          imageId,
	}
}

type Containers []*Container
type ContainerOptions map[string]string

type ContainerStatus struct {
	// uuid
	Id UID
	// name
	Name string
	// nodeName
	NodeName string
	// container state
	ContainerState ContainerState
	// container created
	CreatedAt time.Time
	// container started
	StartedAt time.Time
	// container finished
	FinishedAt time.Time
	// reason of state
	Reason string
	// last message in container
	Message string
	// last error in container
	Error error
}

func NewContainerStatus(id UID, name, nodeName string) *ContainerStatus {
	return &ContainerStatus{
		Id:             id,
		Name:           name,
		NodeName:       nodeName,
		ContainerState: ContainerUnknown,
		CreatedAt:      time.Time{},
		StartedAt:      time.Time{},
		FinishedAt:     time.Time{},
		Reason:         "created by NewContainerStatus",
		Message:        "",
		Error:          nil,
	}
}

type ContainerStatuses []*ContainerStatus
type ContainerState string

const (
	ContainerUnknown ContainerState = "unknown"
	ContainerCreated ContainerState = "created"
	ContainerRunning ContainerState = "running"
	ContainerExited  ContainerState = "exited"
)

type ContainerClient interface{}

//type DefaultContainerClient

// Node is a machine hosting container.
type Node struct {
	// uuid
	Id UID
	// name for human
	Name string
	// current state
	NodeState NodeState
	// container operation client
	Client ContainerClient
	// resource info for provider, not managed by cluster
	ResourceInfo ResourceInfo
	// resource provider
	ResourceProvider ResourceProvider
}

// Status of Node
type NodeStatus struct {
	// uuid
	Id UID
	// name
	Name      string
	Namespace string
	// node state
	NodeState NodeState
	// node created
	CreatedAt time.Time
	// node started
	StartedAt time.Time
	// node finished
	FinishedAt time.Time
	// reason of state
	Reason string
	// last message in node
	Message string
	// last error in node
	Error error
	// Load
	LoadAverage float64
	// memory, MB
	Memory int
	// Disk, GB
	Disk int
}

type Nodes []*Node
type NodeStatuses []*NodeStatus

type NodeState string

const (
	NodeUnknown NodeState = "unknown"
	NodeCreated NodeState = "created"
	NodeRunning NodeState = "running"
	NodeExited  NodeState = "exited"
)

type ResourceInfo map[string]string

type ResourceProvider interface {
	RunNode(*Node) (*ResourceInfo, error)
	StopNode(*Node) error
	RemoveNode(*Node) error
}

// TODO
func (n *Node) RunContainer(container *Container) error {
	//n.Client.Run(contaner)
	return nil
}

func genUID() UID {
	return uuidToUID(uuid.New())
}

func uuidToUID(uuid uuid.UUID) UID {
	var str interface{}
	str = uuid.String()
	return str.(UID)
}
