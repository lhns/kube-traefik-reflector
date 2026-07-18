package controller

// Annotation and label keys, following the kube-vnet convention: user-facing
// *input* keys use the bare `traefik-reflector/` prefix; operator-*owned* output
// keys use the `traefik-reflector.system/` prefix.
const (
	// AnnNamespaces is the input annotation listing target namespaces: a comma
	// separated list, or "*" for all. Absent means the object is not reflected.
	AnnNamespaces = "traefik-reflector/namespaces"
	// AnnNamespaceSelector is an input annotation holding a label selector matched
	// against namespace labels.
	AnnNamespaceSelector = "traefik-reflector/namespace-selector"
	// LabelNamespaceExclude, set to "true" on a Namespace, opts it out of receiving
	// any mirrors.
	LabelNamespaceExclude = "traefik-reflector/exclude"

	// LabelManagedBy is the private ownership sentinel. It is the ONLY key the
	// controller ever selects, sweeps, or deletes on. Never gate GC on the public
	// app.kubernetes.io/managed-by label (users and Helm write that one too).
	LabelManagedBy      = "traefik-reflector.system/managed-by"
	LabelManagedByValue = "traefik-reflector"
	// LabelSourceNamespace / LabelSourceName back-reference the source object and
	// are used to select a source's mirrors for update and garbage collection.
	LabelSourceNamespace = "traefik-reflector.system/source-namespace"
	LabelSourceName      = "traefik-reflector.system/source-name"
	// AnnReflectedFrom records the full "<ns>/<name>" origin (human/observability);
	// AnnSourceUID pins the exact source instance so a delete+recreate of a
	// same-named source is not mistaken for the old one.
	AnnReflectedFrom = "traefik-reflector.system/reflected-from"
	AnnSourceUID     = "traefik-reflector.system/source-uid"

	// LabelK8sManagedBy is the ecosystem-standard managed-by label. INFORMATIONAL
	// ONLY — for kubectl/dashboards. Never select or delete on it.
	LabelK8sManagedBy      = "app.kubernetes.io/managed-by"
	LabelK8sManagedByValue = "traefik-reflector"

	// FieldManager is the server-side-apply field owner for mirrors.
	FieldManager = "traefik-reflector"

	// WildcardNamespaces is the AnnNamespaces value selecting all namespaces.
	WildcardNamespaces = "*"
)

// DefaultExcludedNamespaces are skipped during "*"/selector expansion (but an
// explicitly named namespace is still honored). Overridable via configuration.
var DefaultExcludedNamespaces = []string{"kube-system", "kube-public", "kube-node-lease"}
