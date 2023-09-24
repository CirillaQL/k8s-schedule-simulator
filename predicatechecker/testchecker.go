package predicatechecker

import (
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	scheduler_config_latest "k8s.io/kubernetes/pkg/scheduler/apis/config/latest"
)

// NewTestPredicateChecker builds test version of PredicateChecker.
func NewTestPredicateChecker(clientset *kubernetes.Clientset) (PredicateChecker, error) {
	schedConfig, err := scheduler_config_latest.Default()
	if err != nil {
		return nil, err
	}

	// just call out to NewSchedulerBasedPredicateChecker but use fake kubeClient
	return NewSchedulerBasedPredicateChecker(informers.NewSharedInformerFactory(clientset, 0), schedConfig)
}
