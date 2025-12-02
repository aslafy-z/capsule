// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"

	"github.com/coredns/coredns/plugin/metadata"
)

// Metadata constants used by the Capsule plugin.
const (
	// MetadataPrefix is the prefix for all Capsule metadata keys.
	MetadataPrefix = "capsule/"

	// MetadataNamespace is the metadata key for the source namespace.
	MetadataNamespace = MetadataPrefix + "namespace"

	// MetadataTenant is the metadata key for the source tenant.
	MetadataTenant = MetadataPrefix + "tenant"

	// KubernetesClientNamespace is the metadata key from the kubernetes plugin.
	KubernetesClientNamespace = "kubernetes/client-namespace"
)

// extractMetadata extracts metadata from the context.
// This relies on the CoreDNS metadata plugin being configured upstream.
func extractMetadata(ctx context.Context, key string) string {
	fullKey := MetadataPrefix + key

	if f := metadata.ValueFunc(ctx, fullKey); f != nil {
		return f()
	}

	// Fallback to kubernetes plugin metadata for namespace
	if key == "namespace" {
		if f := metadata.ValueFunc(ctx, KubernetesClientNamespace); f != nil {
			return f()
		}
	}

	return ""
}
