package main

import (
	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
	//appsv1 "k8s.io/api/apps/v1"
	//apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/client-go/kubernetes"
	//"error"
	//"flag"
	"github.com/litmuschaos/kube-helper/kubernetes/container"
	"github.com/litmuschaos/kube-helper/kubernetes/job"
	"github.com/litmuschaos/kube-helper/kubernetes/podtemplatespec"
	"github.com/litmuschaos/litmus/utils/goUtils"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	//"k8s.io/client-go/util/retry"
	"flag"
	"os"
	"strings"
	"time"
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

	//ar s string =
	//config, err := getKubeConfig()
	engine := os.Getenv("CHAOSENGINE")
	experimentList := os.Getenv("EXPERIMENT_LIST")
	appLabel := os.Getenv("APP_LABEL")
	appNamespace := os.Getenv("APP_NAMESPACE")
	appKind := os.Getenv("APP_KIND")
	svcAcc := os.Getenv("CHAOS_SVC_ACC")
	//rand := os.Getenv("RANDOM")
	//max := os.Getenv("MAX_DURATION")
	experiments := strings.Split(experimentList, ",")
	log.Infoln("experiments : ")
	log.Infoln(experiments)
	//fmt.Println(config)
	log.Infoln("engine name : " + engine)
	log.Infoln("AppLabel : " + appLabel)
	log.Infoln("AppNamespace : " + appNamespace)
	//appNamespace = "default"
	log.Infoln("AppKind : " + appKind)
	log.Infoln("Service Account Name : " + svcAcc)
	//svcAcc = "nginx"

	for i := range experiments {
		log.Infoln("Now, going with the experiment Name : " + experiments[i])
		isFound := !goUtils.CheckExperimentInAppNamespace("default", experiments[i], config)
		log.Info("Is the Experiment Found in " + appNamespace + " : ")
		log.Infoln(isFound)
		if !isFound {
			log.Infoln("Not Found Experiment Name : " + experiments[i])
			log.Infoln("breaking the FOR LOOP")
			break
		}
		log.Infoln("Getting the Default ENV Variables")
		env := goUtils.GetList(appNamespace, experiments[i], config)
		log.Info("Printing the Default Variables")
		log.Infoln(env)
		//mt.Println(k)
		log.Infoln("OverWriting the Variables")
		goUtils.OverWriteList(appNamespace, engine, config, env, experiments[i])
		//env has the ENV variables now
		log.Infoln("Patching some required ENV's")
		env["CHAOSENGINE"] = engine
		env["APP_LABEL"] = appLabel
		env["APP_NAMESPACE"] = appNamespace
		env["APP_KIND"] = appKind

		log.Info("Printing the Over-ridden Variables")
		log.Infoln(env)

		//covert env variables to corev1.EnvVars to pass it to builder function
		log.Infoln("Converting the Variables using A Range loop to convert the map of ENV to corev1.EnvVar to directly send to the Builder Func")
		var envVar []corev1.EnvVar
		for k, v := range env {
			var perEnv corev1.EnvVar
			perEnv.Name = k
			perEnv.Value = v
			envVar = append(envVar, perEnv)
		}
		log.Info("Printing the corev1.EnvVar : ")
		log.Infoln(envVar)
		log.Infoln("getting all the details of the experiment Name : " + experiments[i])

		expLabels, expImage, expArgs := goUtils.GetDetails(appNamespace, experiments[i], config)
		log.Infoln("Varibles for Job")
		log.Info("Experiment Labels : ")
		log.Infoln(expLabels)
		log.Info("Experiment Image : ")
		log.Infoln(expImage)
		log.Info("Experiment args : ")
		log.Infoln(expArgs)

		//command := prependString(expArgs, "/bin/bash")
		//randon string generation
		randomString := goUtils.RandomString()

		jobName := experiments[i] + "-" + randomString

		log.Infoln("Printing the JobName with Random String : " + jobName)
		podtemplate := podtemplatespec.NewBuilder().
			WithName(jobName).
			WithNamespace(appNamespace).
			WithLabels(expLabels).
			WithServiceAccountName(svcAcc).
			WithContainerBuilders(
				container.NewBuilder().
					WithName(jobName).
					WithImage(expImage).
					WithCommandNew([]string{"/bin/bash"}).
					WithArgumentsNew(expArgs).
					WithImagePullPolicy("Always").
					WithEnvsNew(envVar),
			)
		restartPolicy := corev1.RestartPolicyOnFailure
		log.Infoln("Job Creation")
		jobObj, err := job.NewBuilder().
			WithName(jobName).
			WithNamespace("default").
			WithLabels(expLabels).
			WithPodTemplateSpecBuilder(podtemplate).
			WithRestartPolicy(restartPolicy).
			Build()

		if err != nil {
			log.Info("Error while building Job : ")
			log.Infoln(err)
		}

		clientSet, litmusClient, err := goUtils.GenerateClientSets(config)
		jobsClient := clientSet.BatchV1().Jobs(appNamespace)
		jobCreationResult, err := jobsClient.Create(jobObj)
		log.Info("Jobcreation", "jobCreation result", jobCreationResult)
		if err != nil {
			log.Info(err)
		}
		var jobStatus int32
		jobStatus = 1
		for jobStatus == 1 {
			log.Infoln("---------------------------------------------------------------------------------------------------")
			expEngine, err := litmusClient.LitmuschaosV1alpha1().ChaosEngines(appNamespace).Get(engine, metav1.GetOptions{})
			if err != nil {
				log.Print(err)
			}
			log.Info(expEngine)
			var currExpStatus v1alpha1.ExperimentStatuses
			currExpStatus.Name = jobName
			currExpStatus.Status = "Running"
			currExpStatus.LastUpdateTime = metav1.Now()
			currExpStatus.Verdict = "Waiting For Completion"
			checkForjobName := checkStatusListForExp(expEngine.Status.Experiments, jobName)
			if checkForjobName == -1 {
				expEngine.Status.Experiments = append(expEngine.Status.Experiments, currExpStatus)
			} else {
				expEngine.Status.Experiments[checkForjobName].LastUpdateTime = metav1.Now()
			}
			log.Info(expEngine)
			_, updateErr := litmusClient.LitmuschaosV1alpha1().ChaosEngines(appNamespace).Update(expEngine)
			if updateErr != nil {
				log.Info("--------------------------------------------")
				log.Info(updateErr)
			}
			getJob, err := clientSet.BatchV1().Jobs(appNamespace).Get(jobName, metav1.GetOptions{})
			if err != nil {
				log.Info("Unable to get the job : ")
				log.Infoln(err)
			}
			jobStatus = getJob.Status.Active
			log.Info("Watching for Job Name : "+jobName+" status of Job : ", jobStatus)
			log.Infoln(jobStatus)
			time.Sleep(5 * time.Second)
		}
		resultName := engine + "-" + experiments[i]
		log.Info("ResultName : " + resultName)
		expResult, err := litmusClient.LitmuschaosV1alpha1().ChaosResults(appNamespace).Get(resultName, metav1.GetOptions{})
		if err != nil {
			log.Infoln("Unable to get result Resource")
			log.Panic(err)
		}
		verdict := expResult.Spec.ExperimentStatus.Verdict
		phase := expResult.Spec.ExperimentStatus.Phase
		expEngine, err := litmusClient.LitmuschaosV1alpha1().ChaosEngines(appNamespace).Get(engine, metav1.GetOptions{})
		if err != nil {
			log.Print(err)
		}
		log.Info(expEngine)
		var currExpStatus v1alpha1.ExperimentStatuses
		currExpStatus.Name = jobName
		currExpStatus.Status = phase
		currExpStatus.LastUpdateTime = metav1.Now()
		currExpStatus.Verdict = verdict
		checkForjobName := checkStatusListForExp(expEngine.Status.Experiments, jobName)
		if checkForjobName == -1 {
			expEngine.Status.Experiments = append(expEngine.Status.Experiments, currExpStatus)
		} else {
			expEngine.Status.Experiments[checkForjobName] = currExpStatus
		}
		//log.Info("--------------------------------")
		log.Info(expEngine)

		_, updateErr := litmusClient.LitmuschaosV1alpha1().ChaosEngines(appNamespace).Update(expEngine)
		if updateErr != nil {
			log.Info("--------------------------------------------")
			log.Info(updateErr)
		}

		if expEngine.Spec.JobCleanUpPolicy == "delete" {
			log.Infoln("Will delete the job as jobCleanPolicy os set to : " + expEngine.Spec.JobCleanUpPolicy)
			deleteJob := clientSet.BatchV1().Jobs(appNamespace).Delete(jobName, &metav1.DeleteOptions{})

			if deleteJob != nil {
				log.Panic(deleteJob)
			}

		}
	}
	//fmt.Println(ans)
}
