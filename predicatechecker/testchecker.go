package predicatechecker

import (
	"k8s.io/client-go/informers"
	clientsetfake "k8s.io/client-go/kubernetes/fake"
	scheduler_config_latest "k8s.io/kubernetes/pkg/scheduler/apis/config/latest"
)

// NewTestPredicateChecker builds test version of PredicateChecker.
func NewTestPredicateChecker() (PredicateChecker, error) {
	schedConfig, err := scheduler_config_latest.Default()
	if err != nil {
		return nil, err
	}

	// just call out to NewSchedulerBasedPredicateChecker but use fake kubeClient
	return NewSchedulerBasedPredicateChecker(informers.NewSharedInformerFactory(clientsetfake.NewSimpleClientset(), 0), schedConfig)
}
