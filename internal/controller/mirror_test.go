package controller

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func sampleSource() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": TraefikGroup + "/" + TraefikVersion,
		"kind":       "Middleware",
		"metadata": map[string]interface{}{
			"name":            "authelia",
			"namespace":       "authelia",
			"uid":             "abc-123",
			"resourceVersion": "9999",
			"generation":      int64(3),
			"annotations":     map[string]interface{}{AnnNamespaces: "duckdb,dashy"},
		},
		"spec": map[string]interface{}{
			"forwardAuth": map[string]interface{}{
				"address":            "http://10.1.2.10:9091/api/authz/forward-auth",
				"trustForwardHeader": true,
			},
		},
		"status": map[string]interface{}{"foo": "bar"},
	}}
}

func TestBuildMirror(t *testing.T) {
	src := sampleSource()
	m := buildMirror(src, "duckdb")

	if m.GetName() != "authelia" || m.GetNamespace() != "duckdb" {
		t.Fatalf("name/namespace = %s/%s, want authelia/duckdb", m.GetName(), m.GetNamespace())
	}
	if m.GroupVersionKind() != src.GroupVersionKind() {
		t.Errorf("gvk = %v, want %v", m.GroupVersionKind(), src.GroupVersionKind())
	}

	// Spec is copied.
	addr, _, _ := unstructured.NestedString(m.Object, "spec", "forwardAuth", "address")
	if addr != "http://10.1.2.10:9091/api/authz/forward-auth" {
		t.Errorf("spec.forwardAuth.address not copied: %q", addr)
	}

	// Server-managed metadata and status are dropped.
	if m.GetUID() != "" || m.GetResourceVersion() != "" || m.GetGeneration() != 0 {
		t.Errorf("server metadata leaked: uid=%q rv=%q gen=%d", m.GetUID(), m.GetResourceVersion(), m.GetGeneration())
	}
	if _, found, _ := unstructured.NestedMap(m.Object, "status"); found {
		t.Error("status should not be copied to mirror")
	}

	// Ownership labels.
	l := m.GetLabels()
	if l[LabelManagedBy] != LabelManagedByValue {
		t.Errorf("missing managed-by sentinel: %v", l)
	}
	if l[LabelSourceNamespace] != "authelia" || l[LabelSourceName] != "authelia" {
		t.Errorf("source labels = %q/%q", l[LabelSourceNamespace], l[LabelSourceName])
	}
	if l[LabelK8sManagedBy] != LabelK8sManagedByValue {
		t.Errorf("missing public managed-by: %v", l)
	}

	// Provenance annotations.
	a := m.GetAnnotations()
	if a[AnnReflectedFrom] != "authelia/authelia" {
		t.Errorf("reflected-from = %q", a[AnnReflectedFrom])
	}
	if a[AnnSourceUID] != "abc-123" {
		t.Errorf("source-uid = %q", a[AnnSourceUID])
	}
	// The source's input annotation must not ride along onto the mirror.
	if _, ok := a[AnnNamespaces]; ok {
		t.Error("mirror should not carry the source targeting annotation")
	}

	// A mirror is recognized as managed; a plain source is not.
	if !isMirror(m) {
		t.Error("isMirror(mirror) = false")
	}
	if isMirror(src) {
		t.Error("isMirror(source) = true")
	}
}

func TestLabelValue(t *testing.T) {
	short := "duckdb"
	if got := labelValue(short); got != short {
		t.Errorf("labelValue(%q) = %q, want unchanged", short, got)
	}

	long := strings.Repeat("a", 80)
	got := labelValue(long)
	if len(got) != 63 {
		t.Errorf("labelValue(long) length = %d, want 63", len(got))
	}
	if !strings.HasPrefix(got, strings.Repeat("a", 54)+"-") {
		t.Errorf("labelValue(long) = %q, want truncated+hash form", got)
	}
	// Deterministic.
	if labelValue(long) != got {
		t.Error("labelValue is not deterministic")
	}
}

func TestSourceSelectorLabels(t *testing.T) {
	got := sourceSelectorLabels("authelia", "authelia")
	if got[LabelManagedBy] != LabelManagedByValue || got[LabelSourceNamespace] != "authelia" || got[LabelSourceName] != "authelia" {
		t.Errorf("sourceSelectorLabels = %v", got)
	}
	if _, ok := got[LabelK8sManagedBy]; ok {
		t.Error("selector labels must not include the public managed-by (never select on it)")
	}
}
