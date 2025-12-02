// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

// Package coredns provides a CoreDNS plugin for Capsule that enforces
// tenant/namespace DNS isolation and supports namespace whitelisting.
//
// The plugin can be configured to run in two isolation modes:
// - Tenant mode: pods can only resolve services within their own tenant
// - Namespace mode: pods can only resolve services within their own namespace
//
// Whitelisted namespaces (e.g., default, kube-system) are always accessible.
package coredns

import (
	"context"
	"errors"
)

// PluginName is the name of the CoreDNS plugin.
const PluginName = "capsule"

// Plugin implements the CoreDNS plugin interface for Capsule DNS isolation.
type Plugin struct {
	config           *Config
	accessController *AccessController
	next             Handler
}

// Handler represents the next plugin in the CoreDNS chain.
type Handler interface {
	ServeDNS(ctx context.Context, request *DNSRequest) (*DNSResponse, error)
}

// DNSRequest represents a DNS query request.
type DNSRequest struct {
	// Question contains the DNS query details.
	Question DNSQuestion

	// SourceIP is the IP address of the requesting pod.
	SourceIP string

	// Metadata contains additional context from CoreDNS.
	Metadata map[string]string
}

// DNSQuestion represents a DNS query question.
type DNSQuestion struct {
	// Name is the queried domain name (FQDN with trailing dot).
	Name string

	// Type is the DNS record type (A, AAAA, SRV, etc.).
	Type uint16
}

// DNSResponse represents a DNS query response.
type DNSResponse struct {
	// Rcode is the DNS response code.
	Rcode int

	// Answer contains the DNS answer records.
	Answer []DNSRecord
}

// DNSRecord represents a single DNS record.
type DNSRecord struct {
	// Name is the record name.
	Name string

	// Type is the record type.
	Type uint16

	// Value is the record value.
	Value string

	// TTL is the time-to-live in seconds.
	TTL uint32
}

// DNS response codes following RFC 1035.
const (
	RcodeSuccess        = 0
	RcodeFormatError    = 1
	RcodeServerFailure  = 2
	RcodeNameError      = 3 // NXDOMAIN
	RcodeNotImplemented = 4
	RcodeRefused        = 5
)

// Common errors.
var (
	ErrAccessDenied    = errors.New("access denied by capsule dns policy")
	ErrNoTenantContext = errors.New("no tenant context available")
)

// NewPlugin creates a new Capsule CoreDNS plugin.
func NewPlugin(config *Config, resolver TenantResolver, next Handler) *Plugin {
	if config == nil {
		config = DefaultConfig()
	}

	return &Plugin{
		config:           config,
		accessController: NewAccessController(config, resolver),
		next:             next,
	}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return PluginName
}

// ServeDNS handles a DNS request and enforces access control.
func (p *Plugin) ServeDNS(ctx context.Context, request *DNSRequest) (*DNSResponse, error) {
	// Extract request context from metadata
	reqCtx := p.extractRequestContext(request)

	// Parse the DNS query to get target namespace
	service, targetNs, ok := ParseDNSQuery(request.Question.Name, "cluster.local")
	if ok {
		reqCtx.TargetNamespace = targetNs
		reqCtx.TargetService = service

		// Check if access is allowed
		allowed, err := p.accessController.IsAccessAllowed(reqCtx)
		if err != nil {
			// On error, deny by default for security
			return &DNSResponse{Rcode: RcodeRefused}, ErrAccessDenied
		}

		if !allowed {
			// Return NXDOMAIN for denied queries to avoid information leakage
			return &DNSResponse{Rcode: RcodeNameError}, nil
		}
	}

	// Access allowed - pass to the next handler
	if p.next != nil {
		return p.next.ServeDNS(ctx, request)
	}

	// No next handler - return server failure
	return &DNSResponse{Rcode: RcodeServerFailure}, nil
}

// extractRequestContext extracts tenant/namespace context from the DNS request metadata.
func (p *Plugin) extractRequestContext(request *DNSRequest) *RequestContext {
	reqCtx := &RequestContext{}

	if request.Metadata != nil {
		// Extract source namespace from metadata
		if ns, ok := request.Metadata["namespace"]; ok {
			reqCtx.SourceNamespace = ns
		}

		// Extract tenant label from metadata
		if tenant, ok := request.Metadata[p.config.TenantLabelKey]; ok {
			reqCtx.SourceTenant = tenant
		}
	}

	return reqCtx
}
