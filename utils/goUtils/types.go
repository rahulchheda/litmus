package goUtils

import (
	//log "github.com/sirupsen/logrus"
	//"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
	clientV1alpha1 "github.com/litmuschaos/chaos-operator/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type EngineDetails struct {
	Name         string
	Experiments  []string
	AppLabel     string
	SvcAccount   string
	AppKind      string
	AppNamespace string
	Config       *rest.Config
}
type ExperimentDetails struct {
	Env       map[string]string
	ExpLabels map[string]string
	ExpImage  string
	ExpArgs   []string
	JobName   string
}
type ClientSets struct {
	KubeClient   *kubernetes.Clientset
	LitmusClient *clientV1alpha1.Clientset
}
