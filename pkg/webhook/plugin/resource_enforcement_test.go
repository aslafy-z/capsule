// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateRequiredLabels(t *testing.T) {
	tests := []struct {
		name           string
		labels         map[string]string
		required       map[string]string
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name:        "no required labels",
			labels:      map[string]string{"app": "test"},
			required:    nil,
			expectedErr: false,
		},
		{
			name:        "all required labels present",
			labels:      map[string]string{"app": "test", "env": "prod"},
			required:    map[string]string{"app": "test"},
			expectedErr: false,
		},
		{
			name:           "missing required label",
			labels:         map[string]string{"app": "test"},
			required:       map[string]string{"env": "prod"},
			expectedErr:    true,
			expectedErrMsg: "missing required label: env",
		},
		{
			name:           "required label with wrong value",
			labels:         map[string]string{"app": "test", "env": "dev"},
			required:       map[string]string{"env": "prod"},
			expectedErr:    true,
			expectedErrMsg: "label env has invalid value: expected prod, got dev",
		},
		{
			name:        "required label with any value",
			labels:      map[string]string{"app": "test", "env": "dev"},
			required:    map[string]string{"env": ""},
			expectedErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRequiredLabels(tc.labels, tc.required)
			if tc.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRequiredAnnotations(t *testing.T) {
	tests := []struct {
		name           string
		annotations    map[string]string
		required       map[string]string
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name:        "no required annotations",
			annotations: map[string]string{"app": "test"},
			required:    nil,
			expectedErr: false,
		},
		{
			name:        "all required annotations present",
			annotations: map[string]string{"app": "test", "description": "test pod"},
			required:    map[string]string{"description": "test pod"},
			expectedErr: false,
		},
		{
			name:           "missing required annotation",
			annotations:    map[string]string{"app": "test"},
			required:       map[string]string{"description": "test pod"},
			expectedErr:    true,
			expectedErrMsg: "missing required annotation: description",
		},
		{
			name:           "required annotation with wrong value",
			annotations:    map[string]string{"app": "test", "description": "dev pod"},
			required:       map[string]string{"description": "prod pod"},
			expectedErr:    true,
			expectedErrMsg: "annotation description has invalid value: expected prod pod, got dev pod",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRequiredAnnotations(tc.annotations, tc.required)
			if tc.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApplyDefaultLabels(t *testing.T) {
	tests := []struct {
		name           string
		pod            *corev1.Pod
		defaults       map[string]string
		expectModified bool
		expectedLabels map[string]string
	}{
		{
			name: "no defaults",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test"}},
			},
			defaults:       nil,
			expectModified: false,
			expectedLabels: map[string]string{"app": "test"},
		},
		{
			name: "apply default to pod without labels",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{},
			},
			defaults:       map[string]string{"env": "prod"},
			expectModified: true,
			expectedLabels: map[string]string{"env": "prod"},
		},
		{
			name: "apply default without overwriting",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test", "env": "dev"}},
			},
			defaults:       map[string]string{"env": "prod", "tier": "frontend"},
			expectModified: true,
			expectedLabels: map[string]string{"app": "test", "env": "dev", "tier": "frontend"},
		},
		{
			name: "no modification when all defaults already exist",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test", "env": "dev"}},
			},
			defaults:       map[string]string{"env": "prod"},
			expectModified: false,
			expectedLabels: map[string]string{"app": "test", "env": "dev"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			modified := applyDefaultLabels(tc.pod, tc.defaults)
			assert.Equal(t, tc.expectModified, modified)
			assert.Equal(t, tc.expectedLabels, tc.pod.Labels)
		})
	}
}

func TestApplyDefaultAnnotations(t *testing.T) {
	tests := []struct {
		name                string
		pod                 *corev1.Pod
		defaults            map[string]string
		expectModified      bool
		expectedAnnotations map[string]string
	}{
		{
			name: "no defaults",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"app": "test"}},
			},
			defaults:            nil,
			expectModified:      false,
			expectedAnnotations: map[string]string{"app": "test"},
		},
		{
			name: "apply default to pod without annotations",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{},
			},
			defaults:            map[string]string{"description": "test pod"},
			expectModified:      true,
			expectedAnnotations: map[string]string{"description": "test pod"},
		},
		{
			name: "apply default without overwriting",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"app": "test", "description": "dev pod"}},
			},
			defaults:            map[string]string{"description": "prod pod", "owner": "team-a"},
			expectModified:      true,
			expectedAnnotations: map[string]string{"app": "test", "description": "dev pod", "owner": "team-a"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			modified := applyDefaultAnnotations(tc.pod, tc.defaults)
			assert.Equal(t, tc.expectModified, modified)
			assert.Equal(t, tc.expectedAnnotations, tc.pod.Annotations)
		})
	}
}

func TestErrorMessages(t *testing.T) {
	t.Run("MissingRequiredLabelError", func(t *testing.T) {
		err := NewMissingRequiredLabelError("env")
		assert.Equal(t, "missing required label: env", err.Error())
	})

	t.Run("InvalidLabelValueError", func(t *testing.T) {
		err := NewInvalidLabelValueError("env", "prod", "dev")
		assert.Equal(t, "label env has invalid value: expected prod, got dev", err.Error())
	})

	t.Run("MissingRequiredAnnotationError", func(t *testing.T) {
		err := NewMissingRequiredAnnotationError("description")
		assert.Equal(t, "missing required annotation: description", err.Error())
	})

	t.Run("InvalidAnnotationValueError", func(t *testing.T) {
		err := NewInvalidAnnotationValueError("description", "prod pod", "dev pod")
		assert.Equal(t, "annotation description has invalid value: expected prod pod, got dev pod", err.Error())
	})
}
