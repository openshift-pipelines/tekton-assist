// Copyright 2025 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	"context"
	"regexp"
	"time"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	k8scache "k8s.io/client-go/tools/cache"
)

// Manager creates and runs SharedIndexInformers for required resources.
type Manager struct {
	taskRunInformer     k8scache.SharedIndexInformer
	pipelineRunInformer k8scache.SharedIndexInformer
	podInformer         k8scache.SharedIndexInformer
}

// NamespaceIgnorePattern matches system namespaces that should be ignored by cache watchers.
// ^(openshift|kube)- matches any namespace beginning with openshift- or kube-
// The rest are exact matches for common system namespaces.
var NamespaceIgnorePattern = regexp.MustCompile("^(openshift|kube)-|^open-cluster-management-agent-addon$|^open-cluster-management-agent$|^dedicated-admin$|^kube-node-lease$|^kube-public$|^kube-system$")

// allowNamespace returns true if the namespace should be included (not ignored).
func allowNamespace(ns string) bool { return !NamespaceIgnorePattern.MatchString(ns) }

// NewManager constructs informers using raw ListWatch to avoid extra deps.
// If namespace is empty, it watches all namespaces.
func NewManager(kube kubernetes.Interface, tekton tektonclient.Interface, namespace string, resync time.Duration) *Manager {
	// Pods
	podLW := &k8scache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			// Filter to Tekton-related Pods only to reduce cardinality
			opts.LabelSelector = withLabelSelector(opts.LabelSelector, "tekton.dev/taskRun")
			list, err := kube.CoreV1().Pods(namespace).List(context.TODO(), opts)
			if err != nil {
				return nil, err
			}
			filtered := make([]corev1.Pod, 0, len(list.Items))
			for _, p := range list.Items {
				if allowNamespace(p.Namespace) {
					filtered = append(filtered, p)
				}
			}
			list.Items = filtered
			return list, nil
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			// Apply the same label selector for watches
			opts.LabelSelector = withLabelSelector(opts.LabelSelector, "tekton.dev/taskRun")
			src, err := kube.CoreV1().Pods(namespace).Watch(context.TODO(), opts)
			if err != nil {
				return nil, err
			}
			return watch.Filter(src, func(e watch.Event) (watch.Event, bool) {
				obj, err := apimeta.Accessor(e.Object)
				if err != nil {
					return e, false
				}
				if allowNamespace(obj.GetNamespace()) {
					return e, true
				}
				return e, false
			}), nil
		},
	}
	podInf := k8scache.NewSharedIndexInformer(podLW, &corev1.Pod{}, resync, k8scache.Indexers{
		k8scache.NamespaceIndex: k8scache.MetaNamespaceIndexFunc,
	})

	// TaskRuns
	trLW := &k8scache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			list, err := tekton.TektonV1().TaskRuns(namespace).List(context.TODO(), opts)
			if err != nil {
				return nil, err
			}
			filtered := make([]pipelinev1.TaskRun, 0, len(list.Items))
			for _, tr := range list.Items {
				if allowNamespace(tr.Namespace) {
					filtered = append(filtered, tr)
				}
			}
			list.Items = filtered
			return list, nil
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			src, err := tekton.TektonV1().TaskRuns(namespace).Watch(context.TODO(), opts)
			if err != nil {
				return nil, err
			}
			return watch.Filter(src, func(e watch.Event) (watch.Event, bool) {
				obj, err := apimeta.Accessor(e.Object)
				if err != nil {
					return e, false
				}
				if allowNamespace(obj.GetNamespace()) {
					return e, true
				}
				return e, false
			}), nil
		},
	}
	trInf := k8scache.NewSharedIndexInformer(trLW, &pipelinev1.TaskRun{}, resync, k8scache.Indexers{
		k8scache.NamespaceIndex: k8scache.MetaNamespaceIndexFunc,
	})

	// PipelineRuns
	prLW := &k8scache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			list, err := tekton.TektonV1().PipelineRuns(namespace).List(context.TODO(), opts)
			if err != nil {
				return nil, err
			}
			filtered := make([]pipelinev1.PipelineRun, 0, len(list.Items))
			for _, pr := range list.Items {
				if allowNamespace(pr.Namespace) {
					filtered = append(filtered, pr)
				}
			}
			list.Items = filtered
			return list, nil
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			src, err := tekton.TektonV1().PipelineRuns(namespace).Watch(context.TODO(), opts)
			if err != nil {
				return nil, err
			}
			return watch.Filter(src, func(e watch.Event) (watch.Event, bool) {
				obj, err := apimeta.Accessor(e.Object)
				if err != nil {
					return e, false
				}
				if allowNamespace(obj.GetNamespace()) {
					return e, true
				}
				return e, false
			}), nil
		},
	}
	prInf := k8scache.NewSharedIndexInformer(prLW, &pipelinev1.PipelineRun{}, resync, k8scache.Indexers{
		k8scache.NamespaceIndex: k8scache.MetaNamespaceIndexFunc,
	})

	return &Manager{
		taskRunInformer:     trInf,
		pipelineRunInformer: prInf,
		podInformer:         podInf,
	}
}

// withLabelSelector appends a label selector term to an existing selector, comma-separated.
// If existing is empty, it returns the added selector.
func withLabelSelector(existing, add string) string {
	if existing == "" {
		return add
	}
	return existing + "," + add
}

// Start runs all informers and waits for initial sync.
func (m *Manager) Start(ctx context.Context) error {
	go m.podInformer.Run(ctx.Done())
	go m.taskRunInformer.Run(ctx.Done())
	go m.pipelineRunInformer.Run(ctx.Done())

	synced := k8scache.WaitForCacheSync(ctx.Done(),
		m.podInformer.HasSynced,
		m.taskRunInformer.HasSynced,
		m.pipelineRunInformer.HasSynced,
	)
	if !synced {
		return context.Canceled
	}
	return nil
}

func (m *Manager) TaskRunInformer() k8scache.SharedIndexInformer     { return m.taskRunInformer }
func (m *Manager) PipelineRunInformer() k8scache.SharedIndexInformer { return m.pipelineRunInformer }
func (m *Manager) PodInformer() k8scache.SharedIndexInformer         { return m.podInformer }
