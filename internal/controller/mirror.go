package controller

import (
	"crypto/sha256"
	"encoding/hex"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// buildMirror returns a fresh mirror of src for targetNS: the spec is deep-copied,
// all server-managed metadata is dropped, and the ownership markers are stamped.
// The result is ready for a server-side-apply.
func buildMirror(src *unstructured.Unstructured, targetNS string) *unstructured.Unstructured {
	m := &unstructured.Unstructured{Object: map[string]interface{}{}}
	m.SetGroupVersionKind(src.GroupVersionKind())
	m.SetName(src.GetName())
	m.SetNamespace(targetNS)

	if spec, found, _ := unstructured.NestedFieldCopy(src.Object, "spec"); found {
		_ = unstructured.SetNestedField(m.Object, spec, "spec")
	}

	m.SetLabels(mirrorLabels(src))
	m.SetAnnotations(map[string]string{
		AnnReflectedFrom: src.GetNamespace() + "/" + src.GetName(),
		AnnSourceUID:     string(src.GetUID()),
	})
	return m
}

// mirrorLabels are the ownership markers stamped on every mirror.
func mirrorLabels(src *unstructured.Unstructured) map[string]string {
	return map[string]string{
		LabelManagedBy:       LabelManagedByValue,
		LabelK8sManagedBy:    LabelK8sManagedByValue,
		LabelSourceNamespace: labelValue(src.GetNamespace()),
		LabelSourceName:      labelValue(src.GetName()),
	}
}

// sourceSelectorLabels are the labels used to list a source's mirrors for update
// and garbage collection.
func sourceSelectorLabels(sourceNS, sourceName string) map[string]string {
	return map[string]string{
		LabelManagedBy:       LabelManagedByValue,
		LabelSourceNamespace: labelValue(sourceNS),
		LabelSourceName:      labelValue(sourceName),
	}
}

// labelValue makes v safe as a label value (<=63 chars). Namespace and object
// names are already RFC 1123 and almost always fit; if v is longer it is truncated
// and a short content hash appended so the value stays unique and valid.
func labelValue(v string) string {
	if len(v) <= 63 {
		return v
	}
	sum := sha256.Sum256([]byte(v))
	return v[:54] + "-" + hex.EncodeToString(sum[:])[:8] // 54 + 1 + 8 = 63
}

// isMirror reports whether obj carries our private ownership sentinel.
func isMirror(obj *unstructured.Unstructured) bool {
	return obj.GetLabels()[LabelManagedBy] == LabelManagedByValue
}
