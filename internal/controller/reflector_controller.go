package controller

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReflectorReconciler mirrors one Traefik CRD (r.Type) across namespaces according
// to the targeting annotations on each source object.
type ReflectorReconciler struct {
	client.Client
	// APIReader is an uncached reader used for the conflict guard so we never miss a
	// freshly-created object that isn't in the cache yet.
	APIReader client.Reader
	Recorder  record.EventRecorder

	Type     ReflectedType
	Excluded map[string]bool

	// SecretExists reports whether a Secret exists in a namespace. In production it
	// reads metadata only (never Secret data). Overridable in tests.
	SecretExists func(ctx context.Context, namespace, name string) (bool, error)
}

// +kubebuilder:rbac:groups=traefik.io,resources=middlewares;middlewaretcps;tlsoptions;serverstransports;serverstransporttcps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile mirrors the source into its target namespaces and prunes stale mirrors.
func (r *ReflectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	src := &unstructured.Unstructured{}
	src.SetGroupVersionKind(r.Type.GVK)
	err := r.Get(ctx, req.NamespacedName, src)
	if apierrors.IsNotFound(err) {
		// Source gone -> delete all its mirrors.
		return ctrl.Result{}, r.deleteMirrors(ctx, req.Namespace, req.Name, nil)
	}
	if err != nil {
		return ctrl.Result{}, err
	}
	// Never treat our own mirror as a source.
	if isMirror(src) {
		return ctrl.Result{}, nil
	}
	// Being deleted -> clean up now (no finalizer; this is best-effort, the orphan
	// sweep is the backstop).
	if src.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, r.deleteMirrors(ctx, req.Namespace, req.Name, nil)
	}

	spec := TargetingSpec{
		Namespaces: src.GetAnnotations()[AnnNamespaces],
		Selector:   src.GetAnnotations()[AnnNamespaceSelector],
	}
	if !spec.HasTargeting() {
		// Not opted in -> ensure no mirrors linger.
		return ctrl.Result{}, r.deleteMirrors(ctx, req.Namespace, req.Name, nil)
	}

	nsInfos, err := r.listNamespaces(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
	targets, err := resolveTargets(req.Namespace, spec, nsInfos, r.Excluded)
	if err != nil {
		// A bad selector is a user error; warn and wait for an edit rather than hot-loop.
		r.Recorder.Eventf(src, corev1.EventTypeWarning, "InvalidSelector",
			"%s: %s is invalid: %v", AnnNamespaceSelector, r.Type.GVK.Kind, err)
		return ctrl.Result{}, nil
	}

	secretNames := referencedSecrets(src, r.Type.SecretRefs)
	desired := make(map[string]bool, len(targets))
	for _, ns := range targets {
		desired[ns] = true
		if err := r.applyMirror(ctx, src, ns, secretNames); err != nil {
			log.Error(err, "apply mirror failed", "targetNamespace", ns)
			return ctrl.Result{}, err
		}
	}

	// Prune mirrors in namespaces no longer targeted.
	return ctrl.Result{}, r.deleteMirrors(ctx, req.Namespace, req.Name, desired)
}

func (r *ReflectorReconciler) applyMirror(ctx context.Context, src *unstructured.Unstructured, targetNS string, secretNames []string) error {
	// Conflict guard: don't clobber a same-named object we don't manage.
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(r.Type.GVK)
	err := r.APIReader.Get(ctx, types.NamespacedName{Namespace: targetNS, Name: src.GetName()}, existing)
	switch {
	case err == nil:
		if !isMirror(existing) {
			r.Recorder.Eventf(src, corev1.EventTypeWarning, "Conflict",
				"a %s named %q already exists in namespace %q and is not managed by %s; not overwriting",
				r.Type.GVK.Kind, src.GetName(), targetNS, LabelManagedByValue)
			return nil
		}
	case apierrors.IsNotFound(err):
		// ok, will create
	default:
		return err
	}

	// Warn (do not block) on missing referenced Secrets — they're the secret
	// reflector's job, not ours.
	for _, sec := range secretNames {
		ok, err := r.secretExists(ctx, targetNS, sec)
		if err != nil {
			return err
		}
		if !ok {
			missingSecretsTotal.WithLabelValues(r.Type.GVK.Kind, targetNS).Inc()
			r.Recorder.Eventf(src, corev1.EventTypeWarning, "MissingSecret",
				"%s references Secret %q absent from namespace %q; reflect the Secret there (mirror still created)",
				r.Type.GVK.Kind, sec, targetNS)
		}
	}

	mirror := buildMirror(src, targetNS)
	return r.Patch(ctx, mirror, client.Apply, client.FieldOwner(FieldManager), client.ForceOwnership)
}

// deleteMirrors deletes this source's mirrors. If keep is non-nil, mirrors in a
// namespace present in keep are retained (used for scope-shrink pruning); a nil
// keep deletes them all (used for source deletion / de-annotation).
func (r *ReflectorReconciler) deleteMirrors(ctx context.Context, srcNS, srcName string, keep map[string]bool) error {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(r.Type.GVK.GroupVersion().WithKind(r.Type.GVK.Kind + "List"))
	if err := r.List(ctx, list, client.MatchingLabels(sourceSelectorLabels(srcNS, srcName))); err != nil {
		return err
	}
	from := srcNS + "/" + srcName
	for i := range list.Items {
		m := &list.Items[i]
		if keep != nil && keep[m.GetNamespace()] {
			continue
		}
		// Guard against label truncation collisions: only act on an exact origin match.
		if m.GetAnnotations()[AnnReflectedFrom] != from {
			continue
		}
		if err := r.Delete(ctx, m); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (r *ReflectorReconciler) listNamespaces(ctx context.Context) ([]NamespaceInfo, error) {
	nsList := &corev1.NamespaceList{}
	if err := r.List(ctx, nsList); err != nil {
		return nil, err
	}
	out := make([]NamespaceInfo, 0, len(nsList.Items))
	for i := range nsList.Items {
		n := &nsList.Items[i]
		out = append(out, NamespaceInfo{
			Name:        n.Name,
			Labels:      n.Labels,
			Terminating: n.Status.Phase == corev1.NamespaceTerminating || n.DeletionTimestamp != nil,
		})
	}
	return out, nil
}

func (r *ReflectorReconciler) secretExists(ctx context.Context, namespace, name string) (bool, error) {
	if r.SecretExists != nil {
		return r.SecretExists(ctx, namespace, name)
	}
	// Metadata-only read: requests PartialObjectMetadata, so Secret data is never fetched.
	meta := &metav1.PartialObjectMetadata{}
	meta.SetGroupVersionKind(schema.GroupVersionKind{Version: "v1", Kind: "Secret"})
	err := r.APIReader.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, meta)
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// SetupWithManager wires the controller: one watch on the GVK (handling both
// sources and our mirrors) plus a namespace watch to fan out on new/changed
// namespaces.
func (r *ReflectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(r.Type.GVK)

	return ctrl.NewControllerManagedBy(mgr).
		Named("reflector-"+strings.ToLower(r.Type.GVK.Kind)).
		Watches(obj, handler.EnqueueRequestsFromMapFunc(r.enqueueForObject)).
		Watches(&corev1.Namespace{}, handler.EnqueueRequestsFromMapFunc(r.enqueueForNamespace),
			builder.WithPredicates(namespaceRelevant())).
		Complete(r)
}

// enqueueForObject maps a watch event to the source key: a mirror maps to its
// origin (via the reflected-from annotation), a source maps to itself.
func (r *ReflectorReconciler) enqueueForObject(_ context.Context, obj client.Object) []reconcile.Request {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil
	}
	if isMirror(u) {
		ns, name, found := strings.Cut(u.GetAnnotations()[AnnReflectedFrom], "/")
		if !found {
			return nil
		}
		return []reconcile.Request{{NamespacedName: types.NamespacedName{Namespace: ns, Name: name}}}
	}
	return []reconcile.Request{{NamespacedName: types.NamespacedName{Namespace: u.GetNamespace(), Name: u.GetName()}}}
}

// enqueueForNamespace re-reconciles every annotated source of this GVK, since a
// new or relabelled namespace may now (or no longer) be a target.
func (r *ReflectorReconciler) enqueueForNamespace(ctx context.Context, _ client.Object) []reconcile.Request {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(r.Type.GVK.GroupVersion().WithKind(r.Type.GVK.Kind + "List"))
	if err := r.List(ctx, list); err != nil {
		return nil
	}
	var reqs []reconcile.Request
	for i := range list.Items {
		it := &list.Items[i]
		if isMirror(it) {
			continue
		}
		spec := TargetingSpec{
			Namespaces: it.GetAnnotations()[AnnNamespaces],
			Selector:   it.GetAnnotations()[AnnNamespaceSelector],
		}
		if !spec.HasTargeting() {
			continue
		}
		reqs = append(reqs, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: it.GetNamespace(), Name: it.GetName()}})
	}
	return reqs
}

// namespaceRelevant fires only on namespace creation or a label change (targeting
// depends on namespace labels; other namespace updates are noise).
func namespaceRelevant() predicate.Predicate {
	return predicate.Or(
		predicate.Funcs{
			CreateFunc:  func(event.CreateEvent) bool { return true },
			DeleteFunc:  func(event.DeleteEvent) bool { return false },
			GenericFunc: func(event.GenericEvent) bool { return false },
			UpdateFunc:  func(event.UpdateEvent) bool { return false },
		},
		predicate.LabelChangedPredicate{},
	)
}
