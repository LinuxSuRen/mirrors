/*
Copyright 2022.

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

package controllers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InvadeReconciler reconciles a Mirror object
type InvadeReconciler struct {
	client.Client
	Log            logr.Logger
	Scheme         *runtime.Scheme
	ConfigFilepath string
}

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create
//+kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="apps",resources=statefulsets,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="apps",resources=daemonsets,verbs=get;list;watch;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Mirror object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.6.4/pkg/reconcile
func (r *InvadeReconciler) Reconcile(cxt context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	log := r.Log.WithValues("invade", req.NamespacedName)

	// load mirror config items
	var items map[string]string
	if items, err = r.loadConfigItems(); err != nil {
		return
	}

	deploy := &appsv1.Deployment{}
	if err = r.Get(cxt, req.NamespacedName, deploy); err != nil {
		err = client.IgnoreNotFound(err)
		return
	}
	log.V(7).Info("process deploy", "nameAndNamespace", req.NamespacedName)

	containers := deploy.Spec.Template.Spec.Containers
	needUpdate := false
	for i, container := range containers {
		if !strings.Contains(container.Image, "@") {
			// only take care of the image with digest
			continue
		}

		for key, item := range items {
			if strings.HasPrefix(container.Image, key) {
				newImg := strings.ReplaceAll(container.Image, key, item)

				// remove digest
				index := strings.Index(newImg, "@")
				newImg = newImg[:index]

				deploy.Spec.Template.Spec.Containers[i].Image = newImg
				needUpdate = true
				break
			}
		}
	}
	if needUpdate {
		log.Info("prepare to update",
			"deploy", fmt.Sprintf("%s/%s", deploy.Namespace, deploy.Name))
		err = r.Update(cxt, deploy)
	}
	return
}

func (r *InvadeReconciler) loadConfigItems() (items map[string]string, err error) {
	items = map[string]string{
		"gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/controller":   "gcriotekton/pipeline-controller",
		"gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/webhook":      "gcriotekton/pipeline-webhook",
		"gcr.io/tekton-releases/github.com/tektoncd/triggers/cmd/controller":   "gcriotekton/triggers-controller",
		"gcr.io/tekton-releases/github.com/tektoncd/triggers/cmd/interceptors": "gcriotekton/triggers-interceptors",
		"gcr.io/tekton-releases/github.com/tektoncd/triggers/cmd/webhook":      "gcriotekton/triggers-webhook",
		"registry.k8s.io/sig-storage":                                          "registry.aliyuncs.com/google_containers",
		"gcr.io/distroless":                                                    "gcriodistroless",
	}

	if r.ConfigFilepath != "" {
		var data []byte
		if data, err = os.ReadFile(r.ConfigFilepath); err == nil {
			err = yaml.Unmarshal(data, &items)
		} else {
			// ignore the error if file not found
			err = nil
		}
	}
	return
}

// SetupWithManager sets up the controller with the Manager.
func (r *InvadeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Complete(r)
}
