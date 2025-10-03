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
	"fmt"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8scache "k8s.io/client-go/tools/cache"
)

// ResourceCache is the read-only cache facade for consumers.
type ResourceCache interface {
	Start(ctx context.Context) error

	GetTaskRun(ctx context.Context, namespace, name string) (*pipelinev1.TaskRun, error)
	ListTaskRuns(ctx context.Context, namespace string, sel labels.Selector) ([]*pipelinev1.TaskRun, error)

	GetPipelineRun(ctx context.Context, namespace, name string) (*pipelinev1.PipelineRun, error)
	ListPipelineRuns(ctx context.Context, namespace string, sel labels.Selector) ([]*pipelinev1.PipelineRun, error)

	GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error)
	ListPods(ctx context.Context, namespace string, sel labels.Selector) ([]*corev1.Pod, error)

	ListTaskRunsForPipelineRun(ctx context.Context, namespace, prName string) ([]*pipelinev1.TaskRun, error)
	ListPodsForTaskRun(ctx context.Context, namespace, trName string) ([]*corev1.Pod, error)
}

type Service struct {
	m *Manager
}

func NewService(m *Manager) *Service { return &Service{m: m} }

func (s *Service) Start(ctx context.Context) error { return s.m.Start(ctx) }

func (s *Service) GetTaskRun(_ context.Context, namespace, name string) (*pipelinev1.TaskRun, error) {
	key := namespace + "/" + name
	obj, exists, err := s.m.TaskRunInformer().GetIndexer().GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("taskrun %s not found in cache", key)
	}
	return obj.(*pipelinev1.TaskRun), nil
}

func (s *Service) ListTaskRuns(_ context.Context, namespace string, sel labels.Selector) ([]*pipelinev1.TaskRun, error) {
	out := []*pipelinev1.TaskRun{}
	err := k8scache.ListAllByNamespace(s.m.TaskRunInformer().GetIndexer(), namespace, sel, func(obj interface{}) {
		out = append(out, obj.(*pipelinev1.TaskRun))
	})
	return out, err
}

func (s *Service) GetPipelineRun(_ context.Context, namespace, name string) (*pipelinev1.PipelineRun, error) {
	key := namespace + "/" + name
	obj, exists, err := s.m.PipelineRunInformer().GetIndexer().GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("pipelinerun %s not found in cache", key)
	}
	return obj.(*pipelinev1.PipelineRun), nil
}

func (s *Service) ListPipelineRuns(_ context.Context, namespace string, sel labels.Selector) ([]*pipelinev1.PipelineRun, error) {
	out := []*pipelinev1.PipelineRun{}
	err := k8scache.ListAllByNamespace(s.m.PipelineRunInformer().GetIndexer(), namespace, sel, func(obj interface{}) {
		out = append(out, obj.(*pipelinev1.PipelineRun))
	})
	return out, err
}

func (s *Service) GetPod(_ context.Context, namespace, name string) (*corev1.Pod, error) {
	key := namespace + "/" + name
	obj, exists, err := s.m.PodInformer().GetIndexer().GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("pod %s not found in cache", key)
	}
	return obj.(*corev1.Pod), nil
}

func (s *Service) ListPods(_ context.Context, namespace string, sel labels.Selector) ([]*corev1.Pod, error) {
	out := []*corev1.Pod{}
	err := k8scache.ListAllByNamespace(s.m.PodInformer().GetIndexer(), namespace, sel, func(obj interface{}) {
		out = append(out, obj.(*corev1.Pod))
	})
	return out, err
}

// Label-based helpers
func (s *Service) ListTaskRunsForPipelineRun(ctx context.Context, namespace, prName string) ([]*pipelinev1.TaskRun, error) {
	selector := labels.SelectorFromSet(labels.Set{"tekton.dev/pipelineRun": prName})
	return s.ListTaskRuns(ctx, namespace, selector)
}

func (s *Service) ListPodsForTaskRun(ctx context.Context, namespace, trName string) ([]*corev1.Pod, error) {
	selector := labels.SelectorFromSet(labels.Set{"tekton.dev/taskRun": trName})
	return s.ListPods(ctx, namespace, selector)
}

// Label-based helpers
// duplicate declarations removed
