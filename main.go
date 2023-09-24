package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/CirillaQL/k8s-schedule-simulator/clustersnapshot"
	"github.com/CirillaQL/k8s-schedule-simulator/predicatechecker"
	"github.com/CirillaQL/k8s-schedule-simulator/scheduling"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	schedulerframework "k8s.io/kubernetes/pkg/scheduler/framework"
	"path/filepath"
	"time"
)

// BuildTestNode creates a node with specified capacity.
func BuildTestNode(name string, millicpu int64, mem int64) *apiv1.Node {
	node := &apiv1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:     name,
			SelfLink: fmt.Sprintf("/api/v1/nodes/%s", name),
			Labels:   map[string]string{"name": name},
		},
		Spec: apiv1.NodeSpec{
			ProviderID: name,
		},
		Status: apiv1.NodeStatus{
			Capacity: apiv1.ResourceList{
				apiv1.ResourcePods: *resource.NewQuantity(100, resource.DecimalSI),
			},
		},
	}

	if millicpu >= 0 {
		node.Status.Capacity[apiv1.ResourceCPU] = *resource.NewMilliQuantity(millicpu, resource.DecimalSI)
	}
	if mem >= 0 {
		node.Status.Capacity[apiv1.ResourceMemory] = *resource.NewQuantity(mem, resource.DecimalSI)
	}

	node.Status.Allocatable = apiv1.ResourceList{}
	for k, v := range node.Status.Capacity {
		node.Status.Allocatable[k] = v
	}

	return node
}

func BuildTestPod(name string, cpu int64, mem int64, options ...func(*apiv1.Pod)) *apiv1.Pod {
	startTime := metav1.Unix(0, 0)
	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:         types.UID(name),
			Namespace:   "default",
			Name:        name,
			SelfLink:    fmt.Sprintf("/api/v1/namespaces/default/pods/%s", name),
			Annotations: map[string]string{},
		},
		Spec: apiv1.PodSpec{
			NodeSelector: map[string]string{"name": "n1"},
			Containers: []apiv1.Container{
				{
					Resources: apiv1.ResourceRequirements{
						Requests: apiv1.ResourceList{},
					},
				},
			},
		},
		Status: apiv1.PodStatus{
			StartTime: &startTime,
		},
	}

	if cpu >= 0 {
		pod.Spec.Containers[0].Resources.Requests[apiv1.ResourceCPU] = *resource.NewMilliQuantity(cpu, resource.DecimalSI)
	}
	if mem >= 0 {
		pod.Spec.Containers[0].Resources.Requests[apiv1.ResourceMemory] = *resource.NewQuantity(mem, resource.DecimalSI)
	}
	for _, o := range options {
		o(pod)
	}
	return pod
}

func SetNodeCondition(node *apiv1.Node, conditionType apiv1.NodeConditionType, status apiv1.ConditionStatus, lastTransition time.Time) {
	for i := range node.Status.Conditions {
		if node.Status.Conditions[i].Type == conditionType {
			node.Status.Conditions[i].LastTransitionTime = metav1.Time{Time: lastTransition}
			node.Status.Conditions[i].Status = status
			return
		}
	}
	// Condition doesn't exist yet.
	condition := apiv1.NodeCondition{
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: metav1.Time{Time: lastTransition},
	}
	node.Status.Conditions = append(node.Status.Conditions, condition)
}

// SetNodeReadyState sets node ready state to either ConditionTrue or ConditionFalse.
func SetNodeReadyState(node *apiv1.Node, ready bool, lastTransition time.Time) {
	if ready {
		SetNodeCondition(node, apiv1.NodeReady, apiv1.ConditionTrue, lastTransition)
	} else {
		SetNodeCondition(node, apiv1.NodeReady, apiv1.ConditionFalse, lastTransition)
		node.Spec.Taints = append(node.Spec.Taints, apiv1.Taint{
			Key:    "node.kubernetes.io/not-ready",
			Value:  "true",
			Effect: apiv1.TaintEffectNoSchedule,
		})
	}
}

func buildReadyNode(name string, cpu, mem int64) *apiv1.Node {
	n := BuildTestNode(name, cpu, mem)
	SetNodeReadyState(n, true, time.Time{})
	return n
}

func buildScheduledPod(name string, cpu, mem int64, nodeName string) *apiv1.Pod {
	p := BuildTestPod(name, cpu, mem)
	p.Spec.NodeName = nodeName
	return p
}

func singleNodeOk(nodeName string) func(*schedulerframework.NodeInfo) bool {
	return func(nodeInfo *schedulerframework.NodeInfo) bool {
		return nodeName == nodeInfo.Node().Name
	}
}

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	clusterSnapshot := clustersnapshot.NewBasicClusterSnapshot()
	predicateChecker, err := predicatechecker.NewTestPredicateChecker(clientset)
	if err != nil {
		panic(err)
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	pods, err := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	podsTest := pods.Items[0]
	fmt.Printf("选中Pod Name: %s \n", podsTest.Name)
	newPods := []*apiv1.Pod{}
	for i := 0; i < 100; i++ {
		newPods = append(newPods, &podsTest)
	}

	clustersnapshot.InitializeClusterSnapshotOrDie(clusterSnapshot, nodes.Items, pods.Items)
	s := scheduling.NewHintingSimulator(predicateChecker)
	statuses, _, err := s.TrySchedulePods(clusterSnapshot, newPods, allTrue, false)
	fmt.Println("调度状态：")
	for _, state := range statuses {
		fmt.Printf("Node Name:  %s \n", state.NodeName)
		fmt.Printf("对应Pod: %s \n", state.Pod.Name)
	}
}

func allTrue(*schedulerframework.NodeInfo) bool {
	return true
}
