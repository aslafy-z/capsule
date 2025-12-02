// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	capsulev1beta2 "github.com/clastix/capsule/api/v1beta2"
	"github.com/clastix/capsule/pkg/api"
	capsulewebhook "github.com/clastix/capsule/pkg/webhook"
	"github.com/clastix/capsule/pkg/webhook/utils"
)

type resourceEnforcement struct{}

func ResourceEnforcement() capsulewebhook.Handler {
	return &resourceEnforcement{}
}

func (r *resourceEnforcement) OnCreate(c client.Client, decoder *admission.Decoder, recorder record.EventRecorder) capsulewebhook.Func {
	return func(ctx context.Context, req admission.Request) *admission.Response {
		return r.handle(ctx, req, c, decoder, recorder)
	}
}

func (r *resourceEnforcement) OnUpdate(c client.Client, decoder *admission.Decoder, recorder record.EventRecorder) capsulewebhook.Func {
	return func(ctx context.Context, req admission.Request) *admission.Response {
		return r.handle(ctx, req, c, decoder, recorder)
	}
}

func (r *resourceEnforcement) OnDelete(_ client.Client, _ *admission.Decoder, _ record.EventRecorder) capsulewebhook.Func {
	return func(_ context.Context, _ admission.Request) *admission.Response {
		return nil
	}
}

func (r *resourceEnforcement) handle(ctx context.Context, req admission.Request, c client.Client, decoder *admission.Decoder, recorder record.EventRecorder) *admission.Response {
	pod := &corev1.Pod{}
	if err := decoder.Decode(req, pod); err != nil {
		return utils.ErroredResponse(err)
	}

	tnt, err := utils.TenantByStatusNamespace(ctx, c, pod.Namespace)
	if err != nil {
		return utils.ErroredResponse(err)
	}

	if tnt == nil {
		return nil
	}

	enforcement := tnt.Spec.ResourceEnforcement
	if enforcement == nil {
		return nil
	}

	// Validate required labels
	if err := validateRequiredLabels(pod.Labels, enforcement.RequiredLabels); err != nil {
		recorder.Eventf(tnt, corev1.EventTypeWarning, "MissingRequiredLabels", "Pod %s/%s is missing required labels: %v", req.Namespace, req.Name, err)
		response := admission.Denied(err.Error())

		return &response
	}

	// Validate required annotations
	if err := validateRequiredAnnotations(pod.Annotations, enforcement.RequiredAnnotations); err != nil {
		recorder.Eventf(tnt, corev1.EventTypeWarning, "MissingRequiredAnnotations", "Pod %s/%s is missing required annotations: %v", req.Namespace, req.Name, err)
		response := admission.Denied(err.Error())

		return &response
	}

	// Apply default labels and annotations if needed
	modified := false
	modified = applyDefaultLabels(pod, enforcement.DefaultLabels) || modified
	modified = applyDefaultAnnotations(pod, enforcement.DefaultAnnotations) || modified

	if modified {
		marshaled, err := json.Marshal(pod)
		if err != nil {
			return utils.ErroredResponse(err)
		}

		recorder.Eventf(tnt, corev1.EventTypeNormal, "DefaultsApplied", "Applied default labels/annotations to Pod %s/%s", pod.Namespace, pod.Name)
		response := admission.PatchResponseFromRaw(req.Object.Raw, marshaled)

		return &response
	}

	return nil
}

func validateRequiredLabels(labels, required map[string]string) error {
	for key, expectedValue := range required {
		value, exists := labels[key]
		if !exists {
			return NewMissingRequiredLabelError(key)
		}

		if expectedValue != "" && value != expectedValue {
			return NewInvalidLabelValueError(key, expectedValue, value)
		}
	}

	return nil
}

func validateRequiredAnnotations(annotations, required map[string]string) error {
	for key, expectedValue := range required {
		value, exists := annotations[key]
		if !exists {
			return NewMissingRequiredAnnotationError(key)
		}

		if expectedValue != "" && value != expectedValue {
			return NewInvalidAnnotationValueError(key, expectedValue, value)
		}
	}

	return nil
}

func applyDefaultLabels(pod *corev1.Pod, defaults map[string]string) bool {
	if len(defaults) == 0 {
		return false
	}

	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}

	modified := false

	for key, value := range defaults {
		if _, exists := pod.Labels[key]; !exists {
			pod.Labels[key] = value
			modified = true
		}
	}

	return modified
}

func applyDefaultAnnotations(pod *corev1.Pod, defaults map[string]string) bool {
	if len(defaults) == 0 {
		return false
	}

	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	modified := false

	for key, value := range defaults {
		if _, exists := pod.Annotations[key]; !exists {
			pod.Annotations[key] = value
			modified = true
		}
	}

	return modified
}

// Enforcement returns a new ResourceEnforcementSpec from the given Tenant.
func Enforcement(tnt *capsulev1beta2.Tenant) *api.ResourceEnforcementSpec {
	return tnt.Spec.ResourceEnforcement
}

// MissingRequiredLabelError is returned when a required label is missing.
type MissingRequiredLabelError struct {
	label string
}

func NewMissingRequiredLabelError(label string) error {
	return &MissingRequiredLabelError{label: label}
}

func (e *MissingRequiredLabelError) Error() string {
	return fmt.Sprintf("missing required label: %s", e.label)
}

// InvalidLabelValueError is returned when a label has an invalid value.
type InvalidLabelValueError struct {
	label    string
	expected string
	actual   string
}

func NewInvalidLabelValueError(label, expected, actual string) error {
	return &InvalidLabelValueError{label: label, expected: expected, actual: actual}
}

func (e *InvalidLabelValueError) Error() string {
	return fmt.Sprintf("label %s has invalid value: expected %s, got %s", e.label, e.expected, e.actual)
}

// MissingRequiredAnnotationError is returned when a required annotation is missing.
type MissingRequiredAnnotationError struct {
	annotation string
}

func NewMissingRequiredAnnotationError(annotation string) error {
	return &MissingRequiredAnnotationError{annotation: annotation}
}

func (e *MissingRequiredAnnotationError) Error() string {
	return fmt.Sprintf("missing required annotation: %s", e.annotation)
}

// InvalidAnnotationValueError is returned when an annotation has an invalid value.
type InvalidAnnotationValueError struct {
	annotation string
	expected   string
	actual     string
}

func NewInvalidAnnotationValueError(annotation, expected, actual string) error {
	return &InvalidAnnotationValueError{annotation: annotation, expected: expected, actual: actual}
}

func (e *InvalidAnnotationValueError) Error() string {
	return fmt.Sprintf("annotation %s has invalid value: expected %s, got %s", e.annotation, e.expected, e.actual)
}
