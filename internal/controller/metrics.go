package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// missingSecretsTotal counts occurrences of a referenced Secret being absent from
// a target namespace at reflection time (the mirror is still created).
var missingSecretsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "traefik_reflector_missing_secret_total",
		Help: "Times a Secret referenced by a reflected Traefik resource was missing in a target namespace.",
	},
	[]string{"kind", "namespace"},
)

func init() {
	metrics.Registry.MustRegister(missingSecretsTotal)
}
