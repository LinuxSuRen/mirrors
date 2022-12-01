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
	"gopkg.in/yaml.v3"
	"os"
	"strings"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MirrorReconciler reconciles a Mirror object
type MirrorReconciler struct {
	client.Client
	Log            logr.Logger
	Scheme         *runtime.Scheme
	ConfigFilepath string
}

const filter = "mirrors"

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Mirror object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.6.4/pkg/reconcile
func (r *MirrorReconciler) Reconcile(cxt context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	_ = r.Log.WithValues("mirror", req.NamespacedName)

	// load mirror config items
	var items map[string]string
	if items, err = r.loadConfigItems(); err != nil {
		return
	}

	pod := &v1.Pod{}
	if err = r.Get(cxt, req.NamespacedName, pod); err != nil {
		err = client.IgnoreNotFound(err)
		return
	}
	if pod.Status.Phase == v1.PodSucceeded {
		return
	}
	if _, ok := pod.Labels[filter]; ok {
		return
	}

	containers := pod.Spec.Containers
	for _, container := range containers {
		if strings.Contains(container.Image, "@") {
			// cannot tag an image with digest
			continue
		}

		var newImg string
		skip := true
		for key, item := range items {
			if strings.HasPrefix(container.Image, key) {
				newImg = strings.ReplaceAll(container.Image, key, item)
				skip = false
				break
			}
		}
		if skip {
			continue
		}

		if r.isPulling(cxt, getID(pod.Spec.NodeName, newImg)) {
			continue
		}

		newPod := &v1.Pod{}
		newPod.Labels = map[string]string{
			filter: getID(pod.Spec.NodeName, newImg),
		}
		newPod.GenerateName = pod.Name
		newPod.Namespace = pod.Namespace
		newPod.Spec.InitContainers = []v1.Container{{
			Name:    "cache",
			Image:   "docker.io/docker:20.10.21-alpine3.16",
			Command: []string{"docker", "pull", newImg},
			VolumeMounts: []v1.VolumeMount{{
				Name:      "sock",
				MountPath: "/var/run/docker.sock",
			}},
		}}
		newPod.Spec.RestartPolicy = v1.RestartPolicyOnFailure
		newPod.Spec.Volumes = []v1.Volume{{
			Name: "sock",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/var/run/docker.sock",
				},
			},
		}}
		newPod.Spec.Containers = []v1.Container{{
			Name:    "tag",
			Image:   "docker.io/docker:20.10.21-alpine3.16",
			Command: []string{"docker", "tag", newImg, container.Image},
			VolumeMounts: []v1.VolumeMount{{
				Name:      "sock",
				MountPath: "/var/run/docker.sock",
			}},
		}}
		newPod.Spec.NodeName = pod.Spec.NodeName
		if err := r.Create(cxt, newPod); err != nil {
			fmt.Println("failed to create pod", err)
		}
	}
	return
}

func getID(nodeName, image string) (result string) {
	result = fmt.Sprintf("%s.%s", nodeName, image)
	result = strings.ReplaceAll(result, "/", "")
	result = strings.ReplaceAll(result, "@", "")
	result = strings.ReplaceAll(result, ":", "")
	return
}

func (r *MirrorReconciler) isPulling(ctx context.Context, name string) (ok bool) {
	podList := &v1.PodList{}
	if err := r.List(ctx, podList, client.MatchingLabels{
		filter: name,
	}); err == nil {
		ok = len(podList.Items) > 0
	}
	return
}

func (r *MirrorReconciler) loadConfigItems() (items map[string]string, err error) {
	items = map[string]string{
		"gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/controller": "gcriotekton/pipeline-controller",
		"gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/webhook":    "gcriotekton/pipeline-webhook",
		"registry.k8s.io/sig-storage":                                        "registry.aliyuncs.com/google_containers",
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
func (r *MirrorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Pod{}).
		Complete(r)
}
