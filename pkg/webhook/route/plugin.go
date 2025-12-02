// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package route

import (
	capsulewebhook "github.com/clastix/capsule/pkg/webhook"
)

// +kubebuilder:webhook:path=/plugin,mutating=true,sideEffects=None,admissionReviewVersions=v1,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=plugin.capsule.clastix.io

type plugin struct {
	handlers []capsulewebhook.Handler
}

func Plugin(handler ...capsulewebhook.Handler) capsulewebhook.Webhook {
	return &plugin{handlers: handler}
}

func (w *plugin) GetHandlers() []capsulewebhook.Handler {
	return w.handlers
}

func (w *plugin) GetPath() string {
	return "/plugin"
}
