package goUtils

import (
	"github.com/litmuschaos/kube-helper/kubernetes/container"
	"github.com/litmuschaos/kube-helper/kubernetes/job"
	"github.com/litmuschaos/kube-helper/kubernetes/podtemplatespec"
	log "github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

type PodTemplateSpec struct {
	Object *corev1.PodTemplateSpec
}
type Builder struct {
	podtemplatespec *PodTemplateSpec
	errs            []error
}

func DeployJob(perExperiment ExperimentDetails, engineDetails EngineDetails, envVar []corev1.EnvVar) error {
	pod := BuildPodTemplateSpec(perExperiment, engineDetails, envVar)
	job, err := BuildJob(pod, perExperiment, engineDetails)
	if err != nil {
		log.Info("Unable to build Job")
		return err
	}
	clientSet, _, err := GenerateClientSets(engineDetails.Config)
	if err != nil {
		log.Info("Unable to generate ClientSet while Creating Job")
		return err
	}
	jobsClient := clientSet.BatchV1().Jobs(engineDetails.AppNamespace)
	jobCreationResult, err := jobsClient.Create(job)
	log.Info("Jobcreation log : ", jobCreationResult)
	if err != nil {
		log.Info("Unable to create the Job with the clientSet")
	}
	return nil
}
func BuildPodTemplateSpec(perExperiment ExperimentDetails, engineDetails EngineDetails, envVar []corev1.EnvVar) *podtemplatespec.Builder {

	podtemplate := podtemplatespec.NewBuilder().
		WithName(perExperiment.JobName).
		WithNamespace(engineDetails.AppNamespace).
		WithLabels(perExperiment.ExpLabels).
		WithServiceAccountName(engineDetails.SvcAccount).
		WithContainerBuilders(
			container.NewBuilder().
				WithName(perExperiment.JobName).
				WithImage(perExperiment.ExpImage).
				WithCommandNew([]string{"/bin/bash"}).
				WithArgumentsNew(perExperiment.ExpArgs).
				WithImagePullPolicy("Always").
				WithEnvsNew(envVar),
		)
	return podtemplate
}

func BuildJob(pod *podtemplatespec.Builder, perExperiment ExperimentDetails, engineDetails EngineDetails) (*batchv1.Job, error) {
	restartPolicy := corev1.RestartPolicyOnFailure
	jobObj, err := job.NewBuilder().
		WithName(perExperiment.JobName).
		WithNamespace(engineDetails.AppNamespace).
		WithLabels(perExperiment.ExpLabels).
		WithPodTemplateSpecBuilder(pod).
		WithRestartPolicy(restartPolicy).
		Build()
	if err != nil {
		return jobObj, err
	}
	return jobObj, nil
}
