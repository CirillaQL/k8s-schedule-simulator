package clustersnapshot

import (
	apiv1 "k8s.io/api/core/v1"
)

func InitializeClusterSnapshotOrDie(
	snapshot ClusterSnapshot,
	nodes []*apiv1.Node,
	pods []*apiv1.Pod) {
	var err error

	snapshot.Clear()

	for _, node := range nodes {
		err = snapshot.AddNode(node)
	}

	for _, pod := range pods {
		if pod.Spec.NodeName != "" {
			err = snapshot.AddPod(pod, pod.Spec.NodeName)
		} else if pod.Status.NominatedNodeName != "" {
			err = snapshot.AddPod(pod, pod.Status.NominatedNodeName)
		} else {
			panic(err)
		}
	}
}
