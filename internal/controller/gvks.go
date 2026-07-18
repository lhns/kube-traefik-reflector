package controller

import "k8s.io/apimachinery/pkg/runtime/schema"

// Traefik CRD group/version we reflect.
const (
	TraefikGroup   = "traefik.io"
	TraefikVersion = "v1alpha1"
)

// SecretRef is a path from an object's root to a field that references one or
// more Kubernetes Secret names.
type SecretRef struct {
	// Path is the sequence of keys from the object root,
	// e.g. []string{"spec", "basicAuth", "secret"}.
	Path []string
	// List is true when the leaf is a []string of Secret names rather than a
	// single string.
	List bool
}

// ReflectedType describes a Traefik CRD the controller mirrors and where, in its
// spec, it references Secrets (so we can warn when those are missing).
type ReflectedType struct {
	GVK        schema.GroupVersionKind
	SecretRefs []SecretRef
}

func traefikGVK(kind string) schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: TraefikGroup, Version: TraefikVersion, Kind: kind}
}

// ReflectedTypes is the authoritative table of the CRDs we mirror. Routers
// (IngressRoute*) are intentionally absent — nothing references them, so they do
// not need reflecting. The SecretRefs must be verified against the supported
// Traefik CRD version (covered by unit tests).
var ReflectedTypes = []ReflectedType{
	{
		GVK: traefikGVK("Middleware"),
		SecretRefs: []SecretRef{
			{Path: []string{"spec", "basicAuth", "secret"}},
			{Path: []string{"spec", "digestAuth", "secret"}},
			{Path: []string{"spec", "forwardAuth", "tls", "caSecret"}},
			{Path: []string{"spec", "forwardAuth", "tls", "certSecret"}},
		},
	},
	{
		// MiddlewareTCP has no Secret references — inherently safe to copy.
		GVK: traefikGVK("MiddlewareTCP"),
	},
	{
		GVK: traefikGVK("TLSOption"),
		SecretRefs: []SecretRef{
			{Path: []string{"spec", "clientAuth", "secretNames"}, List: true},
		},
	},
	{
		GVK: traefikGVK("ServersTransport"),
		SecretRefs: []SecretRef{
			{Path: []string{"spec", "certificatesSecrets"}, List: true},
			{Path: []string{"spec", "rootCAsSecrets"}, List: true},
		},
	},
	{
		GVK: traefikGVK("ServersTransportTCP"),
		SecretRefs: []SecretRef{
			{Path: []string{"spec", "tls", "certificatesSecrets"}, List: true},
			{Path: []string{"spec", "tls", "rootCAsSecrets"}, List: true},
		},
	},
}
