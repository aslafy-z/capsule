// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"context"

	"github.com/coredns/coredns/plugin/metadata"
)

// Metadata keys used by the Capsule plugin.
const (
	// MetadataNamespace is the metadata key for the source namespace.
	MetadataNamespace = "capsule/namespace"

	// MetadataTenant is the metadata key for the source tenant.
	MetadataTenant = "capsule/tenant"
)

// extractMetadata extracts metadata from the context.
// This relies on the CoreDNS metadata plugin being configured upstream.
func extractMetadata(ctx context.Context, key string) string {
	fullKey := "capsule/" + key

	if f := metadata.ValueFunc(ctx, fullKey); f != nil {
		return f()
	}

	// Fallback to kubernetes plugin metadata for namespace
	if key == "namespace" {
		if f := metadata.ValueFunc(ctx, "kubernetes/client-namespace"); f != nil {
			return f()
		}
	}

	return ""
}
