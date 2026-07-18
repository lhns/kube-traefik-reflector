package controller

import (
	"context"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// OrphanSweeper periodically deletes mirrors whose source object no longer exists.
// Because the controller uses no finalizers, a source deleted while the controller
// was down would otherwise leave orphans behind; this is the backstop that closes
// that gap. It runs on the elected leader only.
type OrphanSweeper struct {
	Client   client.Client
	Types    []ReflectedType
	Interval time.Duration
}

var _ manager.Runnable = &OrphanSweeper{}
var _ manager.LeaderElectionRunnable = &OrphanSweeper{}

// NeedLeaderElection ensures only the leader sweeps.
func (s *OrphanSweeper) NeedLeaderElection() bool { return true }

func (s *OrphanSweeper) Start(ctx context.Context) error {
	t := time.NewTicker(s.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			s.sweep(ctx)
		}
	}
}

func (s *OrphanSweeper) sweep(ctx context.Context) {
	log := logf.FromContext(ctx).WithName("orphan-sweep")
	for _, rt := range s.Types {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(rt.GVK.GroupVersion().WithKind(rt.GVK.Kind + "List"))
		if err := s.Client.List(ctx, list, client.MatchingLabels{LabelManagedBy: LabelManagedByValue}); err != nil {
			log.Error(err, "list mirrors", "kind", rt.GVK.Kind)
			continue
		}
		for i := range list.Items {
			m := &list.Items[i]
			ns, name, ok := strings.Cut(m.GetAnnotations()[AnnReflectedFrom], "/")
			if !ok {
				continue
			}
			src := &unstructured.Unstructured{}
			src.SetGroupVersionKind(rt.GVK)
			err := s.Client.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, src)
			if err == nil {
				continue // source still exists; owning reconcile keeps it in sync
			}
			if !apierrors.IsNotFound(err) {
				log.Error(err, "get source", "source", m.GetAnnotations()[AnnReflectedFrom])
				continue
			}
			if delErr := s.Client.Delete(ctx, m); delErr != nil && !apierrors.IsNotFound(delErr) {
				log.Error(delErr, "delete orphan mirror", "mirror", m.GetNamespace()+"/"+m.GetName())
				continue
			}
			log.Info("deleted orphan mirror",
				"mirror", m.GetNamespace()+"/"+m.GetName(),
				"missingSource", m.GetAnnotations()[AnnReflectedFrom])
		}
	}
}
