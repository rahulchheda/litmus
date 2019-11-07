package goUtils

import (
	//"fmt"
	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
	//clientV1alpha1 "github.com/litmuschaos/chaos-operator/pkg/client/clientset/versioned"
	//"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/rest"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func checkStatusListForExp(status []v1alpha1.ExperimentStatuses, jobName string) int {
	for i := range status {
		if status[i].Name == jobName {
			return i
		}
	}
	return -1
}
func WatchingJobtillCompletion(perExperiment ExperimentDetails, engineDetails EngineDetails, clients ClientSets) error {
	var jobStatus int32
	jobStatus = 1
	for jobStatus == 1 {
		log.Infoln("---------------------------------------------------------------------------------------------------")
		expEngine, err := clients.LitmusClient.LitmuschaosV1alpha1().ChaosEngines(engineDetails.AppNamespace).Get(engineDetails.Name, metav1.GetOptions{})
		if err != nil {
			log.Print(err)
			return err
		}
		//log.Info(expEngine)
		var currExpStatus v1alpha1.ExperimentStatuses
		currExpStatus.Name = perExperiment.JobName
		currExpStatus.Status = "Running"
		currExpStatus.LastUpdateTime = metav1.Now()
		currExpStatus.Verdict = "Waiting For Completion"
		checkForjobName := checkStatusListForExp(expEngine.Status.Experiments, perExperiment.JobName)
		if checkForjobName == -1 {
			expEngine.Status.Experiments = append(expEngine.Status.Experiments, currExpStatus)
		} else {
			expEngine.Status.Experiments[checkForjobName].LastUpdateTime = metav1.Now()
		}
		log.Info("Patching Engine")
		_, updateErr := clients.LitmusClient.LitmuschaosV1alpha1().ChaosEngines(engineDetails.AppNamespace).Update(expEngine)
		if updateErr != nil {
			//log.Info("--------------------------------------------")
			//log.Info(updateErr)
			return err
		}
		getJob, err := clients.KubeClient.BatchV1().Jobs(engineDetails.AppNamespace).Get(perExperiment.JobName, metav1.GetOptions{})
		if err != nil {
			log.Infoln("Unable to get the job : ", err)
			return err
		}
		jobStatus = getJob.Status.Active
		log.Infoln("Watching for Job Name : "+perExperiment.JobName+" status of Job : ", jobStatus)
		//log.Infoln(jobStatus)
		time.Sleep(5 * time.Second)
	}
	return nil

}
func GetResultName(engineDetails EngineDetails, i int) string {
	resultName := engineDetails.Name + "-" + engineDetails.Experiments[i]
	log.Info("ResultName : " + resultName)
	return resultName
}

func UpdateResultWithJobAndDeletingJob(engineDetails EngineDetails, clients ClientSets, resultName string, perExperiment ExperimentDetails) error {
	expResult, err := clients.LitmusClient.LitmuschaosV1alpha1().ChaosResults(engineDetails.AppNamespace).Get(resultName, metav1.GetOptions{})
	if err != nil {
		log.Infoln("Unable to get result Resource")
		log.Panic(err)
		return err
	}
	verdict := expResult.Spec.ExperimentStatus.Verdict
	phase := expResult.Spec.ExperimentStatus.Phase
	expEngine, err := clients.LitmusClient.LitmuschaosV1alpha1().ChaosEngines(engineDetails.AppNamespace).Get(engineDetails.Name, metav1.GetOptions{})
	if err != nil {
		log.Print(err)
		return err
	}
	log.Info(expEngine)
	var currExpStatus v1alpha1.ExperimentStatuses
	currExpStatus.Name = perExperiment.JobName
	currExpStatus.Status = phase
	currExpStatus.LastUpdateTime = metav1.Now()
	currExpStatus.Verdict = verdict
	checkForjobName := checkStatusListForExp(expEngine.Status.Experiments, perExperiment.JobName)
	if checkForjobName == -1 {
		expEngine.Status.Experiments = append(expEngine.Status.Experiments, currExpStatus)
	} else {
		expEngine.Status.Experiments[checkForjobName] = currExpStatus
	}
	//log.Info("--------------------------------")
	log.Info(expEngine)
	_, updateErr := clients.LitmusClient.LitmuschaosV1alpha1().ChaosEngines(engineDetails.AppNamespace).Update(expEngine)
	if updateErr != nil {
		log.Info("Updating Resource Error : ", updateErr)
		return updateErr
	}
	if expEngine.Spec.JobCleanUpPolicy == "delete" {
		log.Infoln("Will delete the job as jobCleanPolicy os set to : " + expEngine.Spec.JobCleanUpPolicy)
		deleteJob := clients.KubeClient.BatchV1().Jobs(engineDetails.AppNamespace).Delete(perExperiment.JobName, &metav1.DeleteOptions{})
		if deleteJob != nil {
			log.Panic(deleteJob)
			return deleteJob
		}

	}
	return nil
}
