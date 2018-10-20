package cluster

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

type mockClusterService struct {
	ClusterService
}

type testImageData struct {
	input  string
	output *Image
	error  error
}

func TestNewImage(t *testing.T) {
	dataList := []testImageData{
		testImageData{"image:tag", &Image{"image", "image:tag"}, nil},
		testImageData{":tag", nil, errors.New("name not found")},
		testImageData{"i:", &Image{"i", "i:"}, nil},
	}
	for _, data := range dataList {
		image, err := NewImage(data.input)
		if data.error != nil && err.Error() != data.error.Error() {
			t.Fatalf("err:%v, input:%v", err, data.input)
		}
		expected := data.output
		if !reflect.DeepEqual(expected, image) {
			t.Errorf("want:%v,have:%v", expected, image)
		}
	}
}

var testImage, _ = NewImage("image:tag")
var testClusterService = NewDefaultClusterService("0.0.0", testImage)
var testContainerStatus = NewContainerStatus("id1", "name1", "nodeName1")
var testContiner = &Container{
	Id:       "id1",
	Name:     "name1",
	Hash:     "hash1",
	NodeId:   "node1",
	NodeName: "nodename1",
	Image:    testImage,
	ImageId:  "image1",
	ContainerStatus: &ContainerStatus{
		Id:             "id1",
		Name:           "name1",
		NodeName:       "nodename1",
		ContainerState: ContainerUnknown,
		CreatedAt:      time.Time{},
		StartedAt:      time.Time{},
		FinishedAt:     time.Time{},
		Reason:         "created by NewContainerStatus",
		Message:        "",
		Error:          nil,
	},
	ContainerOptions: map[string]string{"options": "option1"},
}

func TestDefaultClusterService_Version(t *testing.T) {
	version, err := testClusterService.Version()
	if err != nil {
		t.Fatal(err)
	}
	if version != "0.0.0" {
		t.Errorf("%v", version)
	}
}

func TestDefaultClusterService_Image(t *testing.T) {
	image, err := testClusterService.Image()
	if err != nil {
		t.Fatal(err)
	}
	if image != testImage {
		t.Errorf("%v", image)
	}

}

func TestNewDefaultClusterService(t *testing.T) {
	clusterService := NewDefaultClusterService("0.0.0", testImage)
	expected := &DefaultClusterService{
		version:    "0.0.0",
		image:      testImage,
		containers: Containers{},
		containerStatuses: ContainerStatuses{},
		nodes: Nodes{},
		nodeStatuses: NodeStatuses{},
		nodesById: make(map[UID]*Node),
		nodesByName: make(map[string]*Node),
		maxNameI:0,
	}
	if !reflect.DeepEqual(clusterService, expected) {
		t.Errorf("%v, %v", clusterService, expected)
	}

}

func TestDefaultClusterService_Containers(t *testing.T) {
	containers, err := testClusterService.Containers(true)
	if err != nil {
		t.Fatal(err)
	}
	expected := Containers{}
	if !reflect.DeepEqual(expected, containers) {
		t.Errorf("%v,%v", expected, containers)
	}
	containersAlive, err := testClusterService.Containers(false)
	if err != nil {
		t.Fatal(err)
	}
	expectedAlive := Containers{}
	if !reflect.DeepEqual(expectedAlive, containersAlive) {
		t.Errorf("%v,%v", expected, containers)
	}
}

func TestDefaultClusterService_ContainerStatus(t *testing.T) {
	clusterService := NewDefaultClusterService("0.0.0", testImage)
	clusterService.containerStatuses = append(clusterService.containerStatuses, testContainerStatus)
	containerStatus, err := clusterService.ContainerStatus("id1", "", "")
	if err != nil {
		t.Fatal(err)
	}
	expected := &ContainerStatus{
		Id:             "id1",
		Name:           "name1",
		NodeName:       "nodeName1",
		ContainerState: ContainerUnknown,
		CreatedAt:      time.Time{},
		StartedAt:      time.Time{},
		FinishedAt:     time.Time{},
		Reason:         "created by NewContainerStatus",
		Message:        "",
		Error:          nil,
	}
	if !reflect.DeepEqual(expected, containerStatus) {
		t.Errorf("%v,%v", expected, containerStatus)
	}

}

func TestNewContainerStatus(t *testing.T) {
	var id UID
	id = "id1"
	containerStatus := NewContainerStatus(id, "name1", "nodeName1")
	expected := &ContainerStatus{
		Id:             "id1",
		Name:           "name1",
		NodeName:       "nodeName1",
		ContainerState: ContainerUnknown,
		CreatedAt:      time.Time{},
		StartedAt:      time.Time{},
		FinishedAt:     time.Time{},
		Reason:         "created by NewContainerStatus",
		Message:        "",
		Error:          nil,
	}
	if !reflect.DeepEqual(expected, containerStatus) {
		t.Errorf("%v,%v", expected, containerStatus)
	}

}

func TestNewContainer(t *testing.T) {
	var id, nodeId UID
	id = "id1"
	name := "name1"
	hash := "hash1"
	nodeId = "node1"
	nodeName := "nodename1"
	containerOptions := ContainerOptions{"options": "option1"}
	container := NewContainer(id, name, hash, nodeId, nodeName, testImage, "image1", containerOptions)
	expected := &Container{
		Id:       "id1",
		Name:     "name1",
		Hash:     "hash1",
		NodeId:   "node1",
		NodeName: "nodename1",
		Image:    testImage,
		ImageId:  "image1",
		ContainerStatus: &ContainerStatus{
			Id:             "id1",
			Name:           "name1",
			NodeName:       "nodename1",
			ContainerState: ContainerUnknown,
			CreatedAt:      time.Time{},
			StartedAt:      time.Time{},
			FinishedAt:     time.Time{},
			Reason:         "created by NewContainerStatus",
			Message:        "",
			Error:          nil,
		},
		ContainerOptions: map[string]string{"options": "option1"},
	}
	if !reflect.DeepEqual(expected, container) {
		t.Errorf("%v,%v", expected, container)
	}
}
