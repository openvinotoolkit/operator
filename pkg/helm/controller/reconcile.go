// Copyright 2018 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//
// Copyright (c) 2022 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package controller

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	rpb "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	resty "github.com/go-resty/resty/v2"
	"github.com/openvinotoolkit/operator/pkg/helm/internal/diff"
	"github.com/openvinotoolkit/operator/pkg/helm/internal/types"
	"github.com/openvinotoolkit/operator/pkg/helm/release"
)

// blank assignment to verify that HelmOperatorReconciler implements reconcile.Reconciler
var _ reconcile.Reconciler = &HelmOperatorReconciler{}

// ReleaseHookFunc defines a function signature for release hooks.
type ReleaseHookFunc func(*rpb.Release) error

// HelmOperatorReconciler reconciles custom resources as Helm releases.
type HelmOperatorReconciler struct {
	Client          client.Client
	EventRecorder   record.EventRecorder
	GVK             schema.GroupVersionKind
	ManagerFactory  release.ManagerFactory
	ReconcilePeriod time.Duration
	OverrideValues  map[string]string
	releaseHook     ReleaseHookFunc
}

const (
	// uninstallFinalizer is added to CRs so they are cleaned up after uninstalling a release.
	uninstallFinalizer = "helm.sdk.operatorframework.io/uninstall-release"
	// Deprecated: use uninstallFinalizer. This will be removed in operator-sdk v2.0.0.
	uninstallFinalizerLegacy = "uninstall-helm-release"

	helmUpgradeForceAnnotation  = "helm.sdk.operatorframework.io/upgrade-force"
	helmUninstallWaitAnnotation = "helm.sdk.operatorframework.io/uninstall-wait"
)

// Reconcile reconciles the requested resource by installing, updating, or
// uninstalling a Helm release based on the resource's current state. If no
// release changes are necessary, Reconcile will create or patch the underlying
// resources to match the expected release manifest.

func (r HelmOperatorReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) { //nolint:gocyclo
	o := &unstructured.Unstructured{}
	o.SetGroupVersionKind(r.GVK)
	o.SetNamespace(request.Namespace)
	o.SetName(request.Name)
	log := log.WithValues(
		"namespace", o.GetNamespace(),
		"name", o.GetName(),
		"apiVersion", o.GetAPIVersion(),
		"kind", o.GetKind(),
	)
	log.Info("Reconciling")

	err := r.Client.Get(ctx, request.NamespacedName, o)
	if apierrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	}
	if err != nil {
		log.Error(err, "Failed to lookup resource")
		return reconcile.Result{}, err
	}

	// Inject current commit SHA (if auto update enabled) and date to notebook resource if not present already
	if r.GVK.Kind == "Notebook" {
		currentNotebookSpec, ok := o.Object["spec"].(map[string]interface{})
		if !ok {
			err = errors.New("bad CR object")
			log.Error(err, "Received bad CR object. Could not parse spec.")
			return reconcile.Result{}, err
		}

		if _, ok := currentNotebookSpec["commit"]; !ok {
			log.Info("Setting initial notebooks build commit SHA")
			ref, err := getGithubRef(currentNotebookSpec)
			if err == nil {
				currentNotebookSpec["commit"] = ref
			} else {
				log.Info("Could not retrieve the initial commit SHA. Setting to empty string.")
				currentNotebookSpec["commit"] = ""
			}
		}

		if _, ok := currentNotebookSpec["latest_update_date"]; !ok {
			log.Info("Setting initial notebooks build date")
			updateDate := getUpdateDate()
			currentNotebookSpec["latest_update_date"] = updateDate
		}
	}

	manager, err := r.ManagerFactory.NewManager(o, r.OverrideValues)
	if err != nil {
		log.Error(err, "Failed to get release manager")
		return reconcile.Result{}, err
	}

	status := types.StatusFor(o)
	log = log.WithValues("release", manager.ReleaseName())

	if o.GetDeletionTimestamp() != nil {
		if !(controllerutil.ContainsFinalizer(o, uninstallFinalizer) ||
			controllerutil.ContainsFinalizer(o, uninstallFinalizerLegacy)) {

			log.Info("Resource is terminated, skipping reconciliation")
			return reconcile.Result{}, nil
		}

		uninstalledRelease, err := manager.UninstallRelease(ctx)
		if err != nil && !errors.Is(err, driver.ErrReleaseNotFound) {
			log.Error(err, "Failed to uninstall release")
			status.SetCondition(types.HelmAppCondition{
				Type:    types.ConditionReleaseFailed,
				Status:  types.StatusTrue,
				Reason:  types.ReasonUninstallError,
				Message: err.Error(),
			})
			if err := r.updateResourceStatus(ctx, o, status); err != nil {
				log.Error(err, "Failed to update status after uninstall release failure")
			}
			return reconcile.Result{}, err
		}
		status.RemoveCondition(types.ConditionReleaseFailed)

		wait := hasAnnotation(helmUninstallWaitAnnotation, o)
		if errors.Is(err, driver.ErrReleaseNotFound) {
			log.Info("Release not found")
		} else {
			log.Info("Uninstalled release")
			if log.V(0).Enabled() && uninstalledRelease != nil {
				fmt.Println(diff.Generate(uninstalledRelease.Manifest, ""))
			}
			if !wait {
				status.SetCondition(types.HelmAppCondition{
					Type:   types.ConditionDeployed,
					Status: types.StatusFalse,
					Reason: types.ReasonUninstallSuccessful,
				})
				status.DeployedRelease = nil
			}
		}
		if wait {
			status.SetCondition(types.HelmAppCondition{
				Type:    types.ConditionDeployed,
				Status:  types.StatusFalse,
				Reason:  types.ReasonUninstallSuccessful,
				Message: "Waiting until all resources are deleted.",
			})
		}
		if err := r.updateResourceStatus(ctx, o, status); err != nil {
			log.Info("Failed to update CR status")
			return reconcile.Result{}, err
		}

		if wait && status.DeployedRelease != nil && status.DeployedRelease.Manifest != "" {
			log.Info("Uninstall wait")
			isAllResourcesDeleted, err := manager.CleanupRelease(ctx, status.DeployedRelease.Manifest)
			if err != nil {
				log.Error(err, "Failed to cleanup release")
				status.SetCondition(types.HelmAppCondition{
					Type:    types.ConditionReleaseFailed,
					Status:  types.StatusTrue,
					Reason:  types.ReasonUninstallError,
					Message: err.Error(),
				})
				_ = r.updateResourceStatus(ctx, o, status)
				return reconcile.Result{}, err
			}
			if !isAllResourcesDeleted {
				log.Info("Waiting until all resources are deleted")
				return reconcile.Result{RequeueAfter: r.ReconcilePeriod}, nil
			}
			status.RemoveCondition(types.ConditionReleaseFailed)
		}

		log.Info("Removing finalizer")
		controllerutil.RemoveFinalizer(o, uninstallFinalizer)
		controllerutil.RemoveFinalizer(o, uninstallFinalizerLegacy)
		if err := r.updateResource(ctx, o); err != nil {
			log.Info("Failed to remove CR uninstall finalizer")
			return reconcile.Result{}, err
		}

		// Since the client is hitting a cache, waiting for the
		// deletion here will guarantee that the next reconciliation
		// will see that the CR has been deleted and that there's
		// nothing left to do.
		if err := r.waitForDeletion(ctx, o); err != nil {
			log.Info("Failed waiting for CR deletion")
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	status.SetCondition(types.HelmAppCondition{
		Type:   types.ConditionInitialized,
		Status: types.StatusTrue,
	})

	if err := manager.Sync(ctx); err != nil {
		log.Error(err, "Failed to sync release")
		status.SetCondition(types.HelmAppCondition{
			Type:    types.ConditionIrreconcilable,
			Status:  types.StatusTrue,
			Reason:  types.ReasonReconcileError,
			Message: err.Error(),
		})
		if err := r.updateResourceStatus(ctx, o, status); err != nil {
			log.Error(err, "Failed to update status after sync release failure")
		}
		return reconcile.Result{}, err
	}
	status.RemoveCondition(types.ConditionIrreconcilable)

	if !manager.IsInstalled() {
		for k, v := range r.OverrideValues {
			r.EventRecorder.Eventf(o, "Warning", "OverrideValuesInUse",
				"Chart value %q overridden to %q by operator's watches.yaml", k, v)
		}

		err = ValidateNotebook(ctx,r.GVK.Kind, request.Namespace)

		if err != nil {
			log.Error(err, "Failed to install release")
			status.SetCondition(types.HelmAppCondition{
				Type:    types.ConditionReleaseFailed,
				Status:  types.StatusTrue,
				Reason:  types.PreconditionError,
				Message: err.Error(),
			})
			if err := r.updateResourceStatus(ctx, o, status); err != nil {
				log.Error(err, "Failed to update status after install release failure")
			}
			return reconcile.Result{}, err
		}

		installedRelease, err := manager.InstallRelease(ctx)
		if err != nil {
			log.Error(err, "Release failed")
			status.SetCondition(types.HelmAppCondition{
				Type:    types.ConditionReleaseFailed,
				Status:  types.StatusTrue,
				Reason:  types.ReasonInstallError,
				Message: err.Error(),
			})
			if err := r.updateResourceStatus(ctx, o, status); err != nil {
				log.Error(err, "Failed to update status after install release failure")
			}
			return reconcile.Result{}, err
		}
		status.RemoveCondition(types.ConditionReleaseFailed)

		log.V(1).Info("Adding finalizer", "finalizer", uninstallFinalizer)
		controllerutil.AddFinalizer(o, uninstallFinalizer)
		if err := r.updateResource(ctx, o); err != nil {
			log.Info("Failed to add CR uninstall finalizer")
			return reconcile.Result{}, err
		}

		if r.releaseHook != nil {
			if err := r.releaseHook(installedRelease); err != nil {
				log.Error(err, "Failed to run release hook")
				return reconcile.Result{}, err
			}
		}

		log.Info("Installed release")
		if log.V(0).Enabled() {
			fmt.Println(diff.Generate("", installedRelease.Manifest))
		}
		log.V(2).Info("Config values", "values", installedRelease.Config)
		message := ""
		if installedRelease.Info != nil {
			message = installedRelease.Info.Notes
		}
		status.SetCondition(types.HelmAppCondition{
			Type:    types.ConditionDeployed,
			Status:  types.StatusTrue,
			Reason:  types.ReasonInstallSuccessful,
			Message: message,
		})
		status.DeployedRelease = &types.HelmAppRelease{
			Name:     installedRelease.Name,
			Manifest: installedRelease.Manifest,
		}

		if r.GVK.Kind == "ModelServer" {
			status.SetScaling(getReplicasStatus(ctx, manager.ReleaseName(), request.Namespace), manager.ReleaseName())
		}

		err = r.updateResourceStatus(ctx, o, status)
		time.Sleep(time.Second)  // wait 1s to reduce conflicts with concurent updates
		return reconcile.Result{RequeueAfter: r.ReconcilePeriod}, err
	}

	if !(controllerutil.ContainsFinalizer(o, uninstallFinalizer) ||
		controllerutil.ContainsFinalizer(o, uninstallFinalizerLegacy)) {

		log.V(1).Info("Adding finalizer", "finalizer", uninstallFinalizer)
		controllerutil.AddFinalizer(o, uninstallFinalizer)
		if err := r.updateResource(ctx, o); err != nil {
			log.Info("Failed to add CR uninstall finalizer")
			return reconcile.Result{}, err
		}
	}

	if manager.IsUpgradeRequired() {
		for k, v := range r.OverrideValues {
			r.EventRecorder.Eventf(o, "Warning", "OverrideValuesInUse",
				"Chart value %q overridden to %q by operator's watches.yaml", k, v)
		}

		err = ValidateNotebook(ctx,r.GVK.Kind, request.Namespace)

		if err != nil {
			log.Error(err, "Failed to upgrade release")
			status.SetCondition(types.HelmAppCondition{
				Type:    types.ConditionReleaseFailed,
				Status:  types.StatusTrue,
				Reason:  types.PreconditionError,
				Message: err.Error(),
			})
			if err := r.updateResourceStatus(ctx, o, status); err != nil {
				log.Error(err, "Failed to update status after upgrade release failure")
			}
			return reconcile.Result{}, err
		}

		force := hasAnnotation(helmUpgradeForceAnnotation, o)
		log.Info("Starting upgrade")
		previousRelease, upgradedRelease, err := manager.UpgradeRelease(ctx, release.ForceUpgrade(force))
		if err != nil {
			log.Error(err, "Release upgrade failed")
			status.SetCondition(types.HelmAppCondition {
				Type:    types.ConditionReleaseFailed,
				Status:  types.StatusTrue,
				Reason:  types.ReasonUpgradeError,
				Message: err.Error(),
			})
			if err := r.updateResourceStatus(ctx, o, status); err != nil {
				log.Error(err, "Failed to update status after sync release failure")
			}
			return reconcile.Result{}, err
		}
		status.RemoveCondition(types.ConditionReleaseFailed)

		if r.releaseHook != nil {
			if err := r.releaseHook(upgradedRelease); err != nil {
				log.Error(err, "Failed to run release hook")
				return reconcile.Result{}, err
			}
		}

		log.Info("Upgraded release", "force", force)
		if log.V(0).Enabled() {
			fmt.Println(diff.Generate(previousRelease.Manifest, upgradedRelease.Manifest))
		}
		log.V(0).Info("Old Config values", "values", previousRelease.Config)
		log.V(0).Info("New Config values", "values", upgradedRelease.Config)

		message := ""
		if upgradedRelease.Info != nil {
			message = upgradedRelease.Info.Notes
		}
		status.SetCondition(types.HelmAppCondition{
			Type:    types.ConditionDeployed,
			Status:  types.StatusTrue,
			Reason:  types.ReasonUpgradeSuccessful,
			Message: message,
		})
		status.DeployedRelease = &types.HelmAppRelease{
			Name:     upgradedRelease.Name,
			Manifest: upgradedRelease.Manifest,
		}

		if r.GVK.Kind == "ModelServer" {
			status.SetScaling(getReplicasStatus(ctx, manager.ReleaseName(), request.Namespace), manager.ReleaseName())
		}
		log.Info("Updating status after upgrade.")
		err = r.updateResourceStatus(ctx, o, status)

		if r.GVK.Kind == "Notebook" {
			if gitRepositoryUpdateRequired(previousRelease.Config, upgradedRelease.Config) {
				log.Info("New repository or branch detected - updating commit value")
				notebookValues := manager.GetValues()
				ref, _ := getGithubRef(notebookValues) // On error during getting new commit sha, set empty string
				updateCommit(ref, notebookValues, manager, o, ctx, r)
				err = r.updateResourceStatus(ctx, o, status)
				if err != nil {
					log.Error(err, "Failed to update resource status")
				}
			} else if gitCommitUpdateRequired(previousRelease.Config, upgradedRelease.Config) {
				log.Info("New commit detected - deleting BuildConfig to trigger build with new configuration")
				buildConfigName := "openvino-notebooks-" + request.Name
				buildConfigNamespace := request.Namespace
	
				buildConfigObj := &unstructured.Unstructured{}
				buildConfigObj.SetName(buildConfigName)
				buildConfigObj.SetNamespace(buildConfigNamespace)
				buildConfigObj.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   "build.openshift.io",
					Kind:    "BuildConfig",
					Version: "v1",
				})
				err = r.Client.Delete(ctx, buildConfigObj)
				if err != nil {
					log.Error(err, "Failed to delete old BuildConfig resource")
				}
			}
		}

		time.Sleep(time.Second)  // wait 1s to reduce conflicts with concurent updates
		return reconcile.Result{RequeueAfter: r.ReconcilePeriod}, err
	}

	// If a change is made to the CR spec that causes a release failure, a
	// ConditionReleaseFailed is added to the status conditions. If that change
	// is then reverted to its previous state, the operator will stop
	// attempting the release and will resume reconciling. In this case, we
	// need to remove the ConditionReleaseFailed because the failing release is
	// no longer being attempted.
	status.RemoveCondition(types.ConditionReleaseFailed)

	err = ValidateNotebook(ctx,r.GVK.Kind, request.Namespace)

	if err != nil {
		log.Error(err, "Failed to reconcile release")
		status.SetCondition(types.HelmAppCondition{
			Type:    types.ConditionReleaseFailed,
			Status:  types.StatusTrue,
			Reason:  types.PreconditionError,
			Message: err.Error(),
		})
		if err := r.updateResourceStatus(ctx, o, status); err != nil {
			log.Error(err, "Failed to update status after reconcile release failure")
		}
		return reconcile.Result{}, err
	}

	expectedRelease, err := manager.ReconcileRelease(ctx)
	if err != nil {
		log.Error(err, "Failed to reconcile release")
		status.SetCondition(types.HelmAppCondition{
			Type:    types.ConditionIrreconcilable,
			Status:  types.StatusTrue,
			Reason:  types.ReasonReconcileError,
			Message: err.Error(),
		})
		if err := r.updateResourceStatus(ctx, o, status); err != nil {
			log.Error(err, "Failed to update status after reconcile release failure")
		}
		return reconcile.Result{}, err
	}
	status.RemoveCondition(types.ConditionIrreconcilable)

	if r.releaseHook != nil {
		if err := r.releaseHook(expectedRelease); err != nil {
			log.Error(err, "Failed to run release hook")
			return reconcile.Result{}, err
		}
	}

	log.Info("Reconciled release")
	reason := types.ReasonUpgradeSuccessful
	if expectedRelease.Version == 1 {
		reason = types.ReasonInstallSuccessful
	}
	message := ""
	if expectedRelease.Info != nil {
		message = expectedRelease.Info.Notes
	}
	status.SetCondition(types.HelmAppCondition{
		Type:    types.ConditionDeployed,
		Status:  types.StatusTrue,
		Reason:  reason,
		Message: message,
	})
	status.DeployedRelease = &types.HelmAppRelease{
		Name:     expectedRelease.Name,
		Manifest: expectedRelease.Manifest,
	}

	if r.GVK.Kind == "ModelServer" {
		status.SetScaling(getReplicasStatus(ctx, manager.ReleaseName(), request.Namespace), manager.ReleaseName())
	}

	if r.GVK.Kind == "Notebook" {
		notebookValues := manager.GetValues()
		if autoUpdateEnabled(notebookValues) {
			ref, err := getGithubRef(notebookValues)
			if err == nil {
				if isCommitUpdateNeeded(ref, notebookValues) {
					updateCommit(ref, notebookValues, manager, o, ctx, r)
				}
			}
			err = r.updateResourceStatus(ctx, o, status)
			if err != nil {
				log.Error(err, "Failed to update resource status")
			}
			return reconcile.Result{RequeueAfter: getNotebookUpdateTimeframe(notebookValues)}, err
		}
	}

	err = r.updateResourceStatus(ctx, o, status)
	if err != nil {
		log.Error(err, "Failed to update resource status")
	}

	return reconcile.Result{}, err
}

func gitRepositoryUpdateRequired(previousReleaseConfig map[string]interface{}, upgradedReleaseConfig map[string]interface{}) bool {
	gitURIChanged := previousReleaseConfig["git_uri"] != upgradedReleaseConfig["git_uri"]
	gitRefChanged := previousReleaseConfig["git_ref"] != upgradedReleaseConfig["git_ref"]
	if gitURIChanged || gitRefChanged { 
		return true 
	}
	return false
}

func gitCommitUpdateRequired(previousReleaseConfig map[string]interface{}, upgradedReleaseConfig map[string]interface{}) bool {
	gitCommitShaChanged := previousReleaseConfig["commit"] != upgradedReleaseConfig["commit"]
	return gitCommitShaChanged 
}

func getNotebookUpdateTimeframe( values map[string]interface{}) time.Duration {
	if reconcileDurationMultiplier, ok := values["reconcile_duration_multiplier"].(int64); ok { 
		return time.Duration(reconcileDurationMultiplier) * time.Minute 
	}
	// By default, requeue reconcile after 2h
	return time.Duration(120) * time.Minute
}

func getUpdateDate() string {
	currentTime := time.Now()
	date := currentTime.Format("2006_Jan_02")
	return date
}

func isCommitUpdateNeeded(ref string, values map[string]interface{}) bool {
	currentCommit, currentCommitDefined := values["commit"]

	if currentCommitDefined {
		log.Info("Current commit defined. Comparing with latest commit...")
		currentCommitSha, currentCommitOk := currentCommit.(string)
		if !currentCommitOk{
			log.Error(errors.New("bad type"), "Latest commit SHA is not string")
			return false
		}
		if ref == currentCommitSha {
			log.Info("Current commit is the latest commit. Update not required.")
			return false
		}
		log.Info("Detected new latest commit. Update in progress...")
	} else {
		log.Info("Current commit not defined - using latest. Update in progress...")
	}
	return true
}

func updateCommit(ref string, values map[string]interface{}, m release.Manager, o *unstructured.Unstructured, ctx context.Context, r HelmOperatorReconciler){
	updateDate := getUpdateDate()
	newSpec := types.HelmAppSpec{
		"git_uri": values["git_uri"].(string),
		"git_ref": values["git_ref"].(string),
		"auto_update_image": values["auto_update_image"].(bool),
		"reconcile_duration_multiplier": values["reconcile_duration_multiplier"].(int64),
		"commit": ref,
		"latest_update_date": updateDate,
	}
	o.Object["spec"] = newSpec
	err := r.updateResource(ctx, o)
	if err == nil {
		log.Info("Updated notebook commit info")
	} else {
		log.Error(err, "Failed to update commit info")
	}
}

func autoUpdateEnabled(values map[string]interface{}) bool {
	if _, ok := values["auto_update_image"].(bool); !ok  { return false } // auto_update_image key is missing
	if values["auto_update_image"] == false { return false }
	if _, ok := values["git_ref"].(string); !ok {  return false } // git_ref key is missing so the commit update is not possible
	if values["git_ref"] == "" { return false }
	return true
}

func getGithubRef(values map[string]interface{}) (string, error) {
	type GithubResponse struct {
		Sha    string    `json:"sha"`
		Commit interface{} `json:"commit"`
	}

	// Create a Resty Client
	client := resty.New()
	var GithubResponseobj GithubResponse
	uri := "https://github.com/openvinotoolkit/openvino_notebooks"
	if val, ok := values["git_uri"].(string); ok {
		uri = val
	}
	url, err := getAPIUrl(uri, values["git_ref"].(string))
	if err != nil {
		log.Error(err, "Cannot create github api request")
		return "", errors.New("cannot create github api request")
	}
	_, err = client.R().SetResult(&GithubResponseobj).Get(url)
	if err != nil {
		log.Error(err, "Cannot connect to notebook github repository")
		return "", err
	}
	log.Info("Connected to notebooks repository: " + url + "; Latest commit: " + GithubResponseobj.Sha)
	if GithubResponseobj.Sha == "" {
		err = errors.New("empty commit sha")
		log.Error(err, "Cannot retrieve latest commit")
		return "", err
	}
	return GithubResponseobj.Sha, nil
}

func getAPIUrl(uri string, branch string) (string, error) {
	re := regexp.MustCompile(`https://github.com/(.*\/.*)`)
	match := re.FindStringSubmatch(uri)
	if match != nil {
		return "https://api.github.com/repos/" + match[1] + "/commits/" + branch, nil
	}
	return "", errors.New("invalid uri " + uri)
}

func getReplicasStatus(ctx context.Context, releaseName string, namespace string) int {
	labelSelector := "release="+releaseName 
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "Can not get api config")
		return 0
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Error(err, "Can not create a clientset")
		return 0
	}
	listOptions:= metav1.ListOptions{LabelSelector: labelSelector}
	k8sclient := clientset.AppsV1()
	deploy, err := k8sclient.Deployments(namespace).List(ctx, listOptions)
	if err != nil {
		log.Error(err, "Can not list deployments")
		return 0
	}
	if len(deploy.Items) == 0 {
		log.Info("Deployment not created yet")
		return 0
	}
	if len(deploy.Items) > 1 {
		log.Info("Multiple deployments created with labelSelector "+ labelSelector + "Remove the conflicting deployment")
		return 0
	}
	return int(deploy.Items[0].Status.AvailableReplicas)
}

func DeploymentNotInstalled(ctx context.Context, deploymentName string, namespaceName string) bool {
	fieldSelector := "metadata.name=" + deploymentName
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "Can not get api config")
		return true
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Error(err, "Can not create a clientset")
		return true
	}
	listOptions:= metav1.ListOptions{FieldSelector: fieldSelector}
	k8sclient := clientset.AppsV1()
	deploy, err := k8sclient.Deployments(namespaceName).List(ctx, listOptions)
	if err != nil {
		log.Error(err, "Can not list deployments")
		return true
	}
	if len(deploy.Items) == 0 {
		log.Info(deploymentName + " operator is not detected")
		return true
	}
	log.Info(deploymentName + " installation detected")
	return false
}

// Returns error if the release is for Notebook resource without the preconditions met
func ValidateNotebook(ctx context.Context, kind string, namespace string) error {
	if kind != "Notebook"{
		return nil
	}
	if namespace != "redhat-ods-applications"{
		return errors.New("notebook resource should be created in redhat-ods-applications project to integrate notebook image with the jupyter hub")
	}
	if  DeploymentNotInstalled(ctx, "rhods-operator", "redhat-ods-operator") && DeploymentNotInstalled(ctx, "opendatahub-operator", "openshift-operators") {
		return errors.New("RHODS operator or ODH operator is required to deploy the notebook image integration with the JupyterHub")
	}
	return nil
}

// returns the boolean representation of the annotation string
// will return false if annotation is not set
func hasAnnotation(anno string, o *unstructured.Unstructured) bool {
	boolStr := o.GetAnnotations()[anno]
	if boolStr == "" {
		return false
	}
	value := false
	if i, err := strconv.ParseBool(boolStr); err != nil {
		log.Info("Could not parse annotation as a boolean",
			"annotation", anno, "value informed", boolStr)
	} else {
		value = i
	}
	return value
}

func (r HelmOperatorReconciler) updateResource(ctx context.Context, o client.Object) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		return r.Client.Update(ctx, o)
	})
}

func (r HelmOperatorReconciler) updateResourceStatus(ctx context.Context, o *unstructured.Unstructured, status *types.HelmAppStatus) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		o.Object["status"] = status
		return r.Client.Status().Update(ctx, o)
	})
}

func (r HelmOperatorReconciler) waitForDeletion(ctx context.Context, o client.Object) error {
	key := client.ObjectKeyFromObject(o)

	tctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	return wait.PollImmediateUntil(time.Millisecond*10, func() (bool, error) {
		err := r.Client.Get(tctx, key, o)
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
	}, tctx.Done())
}
