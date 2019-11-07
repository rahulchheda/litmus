package main

import (
	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
	//appsv1 "k8s.io/api/apps/v1"
	//apiv1 "k8s.io/api/core/v1"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/client-go/kubernetes"
	//"error"
	//"flag"
	//"github.com/litmuschaos/kube-helper/kubernetes/container"
	//"github.com/litmuschaos/kube-helper/kubernetes/job"
	//"github.com/litmuschaos/kube-helper/kubernetes/podtemplatespec"
	"github.com/litmuschaos/litmus/utils/goUtils"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	//"k8s.io/client-go/util/retry"
	"flag"
	"os"
	"strings"
	//z"time"
)

// getKubeConfig setup the config for access cluster resource
func getKubeConfig() (*rest.Config, error) {
	kubeconfig := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	flag.Parse()
	// Use in-cluster config if kubeconfig path is specified
	if *kubeconfig == "" {
		config, err := rest.InClusterConfig()
		if err != nil {
			return config, err
		}
	}
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return config, err
	}
	return config, err
}
func checkStatusListForExp(status []v1alpha1.ExperimentStatuses, jobName string) int {
	for i := range status {
		if status[i].Name == jobName {
			return i
		}
	}
	return -1
}

var kubeconfig string
var err error
var config *rest.Config

func main() {

	var engineDetails goUtils.EngineDetails
	//flag.StringVar(&kubeconfig, "kubeconfig", "", "path to the kubeconfig file")
	//flag.Parse()
	// if kubeconfig == "" {
	// 	log.Info("using the in-cluster config")
	// 	config, err = rest.InClusterConfig()
	// } else {
	// 	log.Info("using configuration from: ", kubeconfig)
	// 	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	// }
	//kubeconfig = "/home/rahul/.kube/config"
	//config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	config, err := getKubeConfig()
	if err != nil {
		log.Info("error in config")
		panic(err.Error())
	}
	engineDetails.Config = config
	experimentList := os.Getenv("EXPERIMENT_LIST")

	//config, err := getKubeConfig()
	engineDetails.Name = os.Getenv("CHAOSENGINE")
	engineDetails.AppLabel = os.Getenv("APP_LABEL")
	engineDetails.AppNamespace = os.Getenv("APP_NAMESPACE")
	engineDetails.AppKind = os.Getenv("APP_KIND")
	engineDetails.SvcAccount = os.Getenv("CHAOS_SVC_ACC")
	engineDetails.Experiments = strings.Split(experimentList, ",")
	//rand := os.Getenv("RANDOM")
	//max := os.Getenv("MAX_DURATION")
	log.Infoln("Experiments List: ", engineDetails.Experiments, " ", "Engine Name: ", engineDetails.Name, " ", "appLabels : ", engineDetails.AppLabel, " ", "appNamespace: ", engineDetails.AppNamespace, " ", "appKind: ", engineDetails.AppKind, " ", "Service Account Name: ", engineDetails.SvcAccount)
	//log.Infoln("experiments : ")
	//log.Infoln(experiments)
	//fmt.Println(config)
	//log.Infoln("engine name : " + engine)
	//log.Infoln("AppLabel : " + appLabel)
	//log.Infoln("AppNamespace : " + appNamespace)
	//appNamespace = "default"
	//log.Infoln("AppKind : " + appKind)
	//log.Infoln("Service Account Name : " + svcAcc)
	//svcAcc = "nginx"

	for i := range engineDetails.Experiments {
		log.Infoln("Going with the experiment Name : " + engineDetails.Experiments[i])
		isFound := !goUtils.CheckExperimentInAppNamespace("default", engineDetails.Experiments[i], config)
		//log.Info("Is the Experiment Found in " + appNamespace + " : ")
		log.Infoln("Experiment Found Status : ", isFound)
		if !isFound {
			log.Infoln("Can't Find Experiment Name : "+engineDetails.Experiments[i], "In Namespace : "+engineDetails.AppNamespace)
			log.Infoln("Not Executing the Experiment : " + engineDetails.Experiments[i])
			break
		}
		var perExperiment goUtils.ExperimentDetails
		//perExperiment.EngineDetails = engineDetails
		log.Infoln("Getting the Default ENV Variables")
		perExperiment.Env = goUtils.GetList(engineDetails.AppNamespace, engineDetails.Experiments[i], engineDetails.Config)
		log.Info("Printing the Default Variables", perExperiment.Env)
		//mt.Println(k)
		log.Infoln("OverWriting the Default Variables")
		goUtils.OverWriteList(engineDetails.AppNamespace, engineDetails.Name, engineDetails.Config, perExperiment.Env, engineDetails.Experiments[i])
		//env has the ENV variables now
		log.Infoln("Patching some required ENV's")
		perExperiment.Env["CHAOSENGINE"] = engineDetails.Name
		perExperiment.Env["APP_LABEL"] = engineDetails.AppLabel
		perExperiment.Env["APP_NAMESPACE"] = engineDetails.AppNamespace
		perExperiment.Env["APP_KIND"] = engineDetails.AppKind

		log.Info("Printing the Over-ridden Variables")
		log.Infoln(perExperiment.Env)

		//covert env variables to corev1.EnvVars to pass it to builder function
		log.Infoln("Converting the Variables using A Range loop to convert the map of ENV to corev1.EnvVar to directly send to the Builder Func")
		var envVar []corev1.EnvVar
		for k, v := range perExperiment.Env {
			var perEnv corev1.EnvVar
			perEnv.Name = k
			perEnv.Value = v
			envVar = append(envVar, perEnv)
		}
		log.Info("Printing the corev1.EnvVar : ")
		log.Infoln(envVar)
		log.Infoln("getting all the details of the experiment Name : " + engineDetails.Experiments[i])

		perExperiment.ExpLabels, perExperiment.ExpImage, perExperiment.ExpArgs = goUtils.GetDetails(engineDetails.AppNamespace, engineDetails.Experiments[i], engineDetails.Config)

		log.Infoln("Variables for ChaosJob : ", "Experiment Labels : ", perExperiment.ExpLabels, " Experiment Image : ", perExperiment.ExpImage, " Experiment Args : ", perExperiment.ExpArgs)

		//command := prependString(expArgs, "/bin/bash")
		//randon string generation
		randomString := goUtils.RandomString()

		perExperiment.JobName = engineDetails.Experiments[i] + "-" + randomString

		log.Infoln("JobName for this Experiment : " + perExperiment.JobName)
		err = goUtils.DeployJob(perExperiment, engineDetails, envVar)
		if err != nil {
			log.Infoln("Error while building Job : ", err)
		}
		var clients goUtils.ClientSets
		clients.KubeClient, clients.LitmusClient, err = goUtils.GenerateClientSets(engineDetails.Config)
		if err != nil {
			log.Info("Unable to generate ClientSet while Creating Job")
			log.Fatal("Unable to create Client Set : ", err)
		}
		resultName := goUtils.GetResultName(engineDetails, i)
		err = goUtils.WatchingJobtillCompletion(perExperiment, engineDetails, clients)
		if err != nil {
			log.Info("Unable to Watch the Job")
			log.Error(err)
		}
		err = goUtils.UpdateResultWithJobAndDeletingJob(engineDetails, clients, resultName, perExperiment)
		if err != nil {
			log.Info("Unable to Update Resource")
			log.Error(err)
		}
	}
	//fmt.Println(ans)
}
