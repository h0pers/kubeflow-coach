/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	coachv1alpha1 "kubeflow-coach/api/v1alpha1"
)

// TrainJobReconciler reconciles a TrainJob object.
type TrainJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=coach.kubeflow.io,resources=trainjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coach.kubeflow.io,resources=trainjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coach.kubeflow.io,resources=trainjobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=coach.kubeflow.io,resources=clustertrainingruntimes,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

func (r *TrainJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Step 1: Get the TrainJob.
	var trainJob coachv1alpha1.TrainJob
	if err := r.Get(ctx, req.NamespacedName, &trainJob); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Step 2: Get the referenced ClusterTrainingRuntime.
	var rt coachv1alpha1.ClusterTrainingRuntime
	if err := r.Get(ctx, client.ObjectKey{Name: trainJob.Spec.RuntimeRef.Name}, &rt); err != nil {
		log.Error(err, "failed to get ClusterTrainingRuntime", "name", trainJob.Spec.RuntimeRef.Name)
		return ctrl.Result{}, r.setCondition(ctx, &trainJob, coachv1alpha1.TrainJobFailed, metav1.ConditionTrue,
			"RuntimeNotFound", fmt.Sprintf("ClusterTrainingRuntime %q not found", trainJob.Spec.RuntimeRef.Name))
	}

	// Step 3: Build the Job from runtime template + TrainJob overrides.
	job := r.buildJob(&trainJob, &rt)
	if err := ctrl.SetControllerReference(&trainJob, job, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Step 4: Create the Job if it doesn't exist.
	var existingJob batchv1.Job
	err := r.Get(ctx, client.ObjectKeyFromObject(job), &existingJob)
	if apierrors.IsNotFound(err) {
		if err := r.setCondition(ctx, &trainJob, coachv1alpha1.TrainJobCreated, metav1.ConditionTrue,
			"JobCreated", "Kubernetes Job created"); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("creating Job", "name", job.Name)
		if err := r.Create(ctx, job); err != nil {
			if apierrors.IsAlreadyExists(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// Step 5: Handle suspend/resume.
	suspend := trainJob.Spec.Suspend != nil && *trainJob.Spec.Suspend
	if existingJob.Spec.Suspend == nil || *existingJob.Spec.Suspend != suspend {
		existingJob.Spec.Suspend = &suspend
		if err := r.Update(ctx, &existingJob); err != nil {
			return ctrl.Result{}, err
		}
		if suspend {
			return ctrl.Result{}, r.setCondition(ctx, &trainJob, coachv1alpha1.TrainJobSuspended, metav1.ConditionTrue,
				"Suspended", "TrainJob suspended")
		}
		return ctrl.Result{}, r.setCondition(ctx, &trainJob, coachv1alpha1.TrainJobSuspended, metav1.ConditionFalse,
			"Resumed", "TrainJob resumed")
	}

	// Step 6: Update TrainJob status from the Job's status.
	return ctrl.Result{}, r.updateStatus(ctx, &trainJob, &existingJob)
}

func (r *TrainJobReconciler) buildJob(trainJob *coachv1alpha1.TrainJob, rt *coachv1alpha1.ClusterTrainingRuntime) *batchv1.Job {
	podSpec := rt.Spec.Template.Spec.DeepCopy()

	if len(podSpec.Containers) > 0 {
		c := &podSpec.Containers[0]
		if trainJob.Spec.Image != nil {
			c.Image = *trainJob.Spec.Image
		}
		if trainJob.Spec.Command != nil {
			c.Command = trainJob.Spec.Command
		}
		if trainJob.Spec.Args != nil {
			c.Args = trainJob.Spec.Args
		}
		if trainJob.Spec.Env != nil {
			c.Env = append(c.Env, trainJob.Spec.Env...)
		}
	}

	numNodes := ptr.Deref(rt.Spec.NumNodes, 1)
	suspend := trainJob.Spec.Suspend != nil && *trainJob.Spec.Suspend

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      trainJob.Name,
			Namespace: trainJob.Namespace,
		},
		Spec: batchv1.JobSpec{
			Parallelism: &numNodes,
			Completions: &numNodes,
			Suspend:     &suspend,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: rt.Spec.Template.ObjectMeta,
				Spec:       *podSpec,
			},
		},
	}
}

func (r *TrainJobReconciler) updateStatus(ctx context.Context, trainJob *coachv1alpha1.TrainJob, job *batchv1.Job) error {
	trainJob.Status.Active = &job.Status.Active
	trainJob.Status.Succeeded = &job.Status.Succeeded
	trainJob.Status.Failed = &job.Status.Failed

	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
			if err := r.setCondition(ctx, trainJob, coachv1alpha1.TrainJobComplete, metav1.ConditionTrue,
				"Complete", "Training completed successfully"); err != nil {
				return err
			}
		}
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			if err := r.setCondition(ctx, trainJob, coachv1alpha1.TrainJobFailed, metav1.ConditionTrue,
				"Failed", c.Message); err != nil {
				return err
			}
		}
	}

	return r.Status().Update(ctx, trainJob)
}

func (r *TrainJobReconciler) setCondition(ctx context.Context, trainJob *coachv1alpha1.TrainJob, condType string, status metav1.ConditionStatus, reason, message string) error {
	changed := false
	for i, c := range trainJob.Status.Conditions {
		if c.Type == condType {
			if c.Status != status || c.Reason != reason || c.Message != message {
				trainJob.Status.Conditions[i].Status = status
				trainJob.Status.Conditions[i].Reason = reason
				trainJob.Status.Conditions[i].Message = message
				trainJob.Status.Conditions[i].LastTransitionTime = metav1.Now()
				changed = true
			}
			if !changed {
				return nil
			}
			return r.Status().Update(ctx, trainJob)
		}
	}
	trainJob.Status.Conditions = append(trainJob.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
	return r.Status().Update(ctx, trainJob)
}

// SetupWithManager tells controller-runtime to watch TrainJobs and Jobs we own.
func (r *TrainJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coachv1alpha1.TrainJob{}).
		Owns(&batchv1.Job{}).
		Named("trainjob").
		Complete(r)
}
