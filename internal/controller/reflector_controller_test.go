package controller

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func middlewareType() ReflectedType {
	for _, rt := range ReflectedTypes {
		if rt.GVK.Kind == "Middleware" {
			return rt
		}
	}
	panic("Middleware type not found")
}

func testScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = corev1.AddToScheme(s)
	gvk := traefikGVK("Middleware")
	s.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
	s.AddKnownTypeWithName(gvk.GroupVersion().WithKind("MiddlewareList"), &unstructured.UnstructuredList{})
	return s
}

func namespace(name string, labels map[string]string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels},
		Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
	}
}

func middleware(ns, name string, ann map[string]string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": TraefikGroup + "/" + TraefikVersion,
		"kind":       "Middleware",
		"metadata":   map[string]interface{}{"name": name, "namespace": ns},
		"spec": map[string]interface{}{
			"forwardAuth": map[string]interface{}{"address": "http://10.1.2.10:9091/api/authz/forward-auth"},
		},
	}}
	if ann != nil {
		u.SetAnnotations(ann)
	}
	return u
}

func newReconciler(objs ...client.Object) (*ReflectorReconciler, client.Client, *record.FakeRecorder) {
	c := fake.NewClientBuilder().WithScheme(testScheme()).WithObjects(objs...).Build()
	rec := record.NewFakeRecorder(50)
	r := &ReflectorReconciler{
		Client:       c,
		APIReader:    c,
		Recorder:     rec,
		Type:         middlewareType(),
		Excluded:     map[string]bool{"kube-system": true, "kube-public": true, "kube-node-lease": true},
		SecretExists: func(context.Context, string, string) (bool, error) { return true, nil },
	}
	return r, c, rec
}

func reconcileSource(t *testing.T, r *ReflectorReconciler, ns, name string) {
	t.Helper()
	if _, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Namespace: ns, Name: name},
	}); err != nil {
		t.Fatalf("Reconcile(%s/%s): %v", ns, name, err)
	}
}

func getMirror(t *testing.T, c client.Client, ns, name string) *unstructured.Unstructured {
	t.Helper()
	m := &unstructured.Unstructured{}
	m.SetGroupVersionKind(traefikGVK("Middleware"))
	err := c.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: name}, m)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("get %s/%s: %v", ns, name, err)
	}
	return m
}

func drainEvents(rec *record.FakeRecorder) []string {
	var out []string
	for {
		select {
		case e := <-rec.Events:
			out = append(out, e)
		default:
			return out
		}
	}
}

func TestReconcile_CreatesMirrors(t *testing.T) {
	src := middleware("authelia", "authelia", map[string]string{AnnNamespaces: "duckdb,dashy"})
	r, c, _ := newReconciler(src, namespace("authelia", nil), namespace("duckdb", nil), namespace("dashy", nil))

	reconcileSource(t, r, "authelia", "authelia")

	for _, ns := range []string{"duckdb", "dashy"} {
		m := getMirror(t, c, ns, "authelia")
		if m == nil {
			t.Fatalf("mirror missing in %s", ns)
		}
		if m.GetLabels()[LabelManagedBy] != LabelManagedByValue {
			t.Errorf("%s mirror missing sentinel label", ns)
		}
		if m.GetAnnotations()[AnnReflectedFrom] != "authelia/authelia" {
			t.Errorf("%s mirror reflected-from = %q", ns, m.GetAnnotations()[AnnReflectedFrom])
		}
		addr, _, _ := unstructured.NestedString(m.Object, "spec", "forwardAuth", "address")
		if addr == "" {
			t.Errorf("%s mirror spec not copied", ns)
		}
	}
}

func TestReconcile_PrunesOnScopeShrink(t *testing.T) {
	src := middleware("authelia", "authelia", map[string]string{AnnNamespaces: "duckdb,dashy"})
	r, c, _ := newReconciler(src, namespace("authelia", nil), namespace("duckdb", nil), namespace("dashy", nil))
	reconcileSource(t, r, "authelia", "authelia")

	// Shrink scope to duckdb only.
	cur := &unstructured.Unstructured{}
	cur.SetGroupVersionKind(traefikGVK("Middleware"))
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "authelia", Name: "authelia"}, cur); err != nil {
		t.Fatal(err)
	}
	cur.SetAnnotations(map[string]string{AnnNamespaces: "duckdb"})
	if err := c.Update(context.Background(), cur); err != nil {
		t.Fatal(err)
	}
	reconcileSource(t, r, "authelia", "authelia")

	if getMirror(t, c, "duckdb", "authelia") == nil {
		t.Error("duckdb mirror should remain")
	}
	if getMirror(t, c, "dashy", "authelia") != nil {
		t.Error("dashy mirror should have been pruned")
	}
}

func TestReconcile_GCOnSourceDeleted(t *testing.T) {
	src := middleware("authelia", "authelia", map[string]string{AnnNamespaces: "duckdb"})
	r, c, _ := newReconciler(src, namespace("authelia", nil), namespace("duckdb", nil))
	reconcileSource(t, r, "authelia", "authelia")
	if getMirror(t, c, "duckdb", "authelia") == nil {
		t.Fatal("precondition: mirror should exist")
	}

	// Delete the source, then reconcile its (now-absent) key.
	if err := c.Delete(context.Background(), src); err != nil {
		t.Fatal(err)
	}
	reconcileSource(t, r, "authelia", "authelia")

	if getMirror(t, c, "duckdb", "authelia") != nil {
		t.Error("mirror should be GC'd after source deletion")
	}
}

func TestReconcile_ConflictSkipsUnmanaged(t *testing.T) {
	src := middleware("authelia", "authelia", map[string]string{AnnNamespaces: "duckdb"})
	// A user's own, unmanaged Middleware of the same name already lives in duckdb.
	userOwned := middleware("duckdb", "authelia", nil)
	unstructured.SetNestedField(userOwned.Object, "USER-OWNED", "spec", "forwardAuth", "address")

	r, c, rec := newReconciler(src, userOwned, namespace("authelia", nil), namespace("duckdb", nil))
	reconcileSource(t, r, "authelia", "authelia")

	m := getMirror(t, c, "duckdb", "authelia")
	if m == nil {
		t.Fatal("object vanished")
	}
	if m.GetLabels()[LabelManagedBy] == LabelManagedByValue {
		t.Error("must NOT overwrite the user's unmanaged resource")
	}
	addr, _, _ := unstructured.NestedString(m.Object, "spec", "forwardAuth", "address")
	if addr != "USER-OWNED" {
		t.Errorf("user resource was overwritten: address=%q", addr)
	}
	if !hasEvent(drainEvents(rec), "Conflict") {
		t.Error("expected a Conflict warning event")
	}
}

func TestReconcile_MissingSecretWarnsButMirrors(t *testing.T) {
	src := middleware("authelia", "authelia", map[string]string{AnnNamespaces: "duckdb"})
	unstructured.SetNestedField(src.Object, "htpasswd", "spec", "basicAuth", "secret")

	r, c, rec := newReconciler(src, namespace("authelia", nil), namespace("duckdb", nil))
	r.SecretExists = func(context.Context, string, string) (bool, error) { return false, nil } // secret absent

	reconcileSource(t, r, "authelia", "authelia")

	if getMirror(t, c, "duckdb", "authelia") == nil {
		t.Error("mirror should still be created despite the missing secret")
	}
	if !hasEvent(drainEvents(rec), "MissingSecret") {
		t.Error("expected a MissingSecret warning event")
	}
}

func hasEvent(events []string, reason string) bool {
	for _, e := range events {
		if strings.Contains(e, reason) {
			return true
		}
	}
	return false
}
