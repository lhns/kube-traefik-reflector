package controller

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func refsForKind(kind string) []SecretRef {
	for _, rt := range ReflectedTypes {
		if rt.GVK.Kind == kind {
			return rt.SecretRefs
		}
	}
	return nil
}

func TestReferencedSecrets(t *testing.T) {
	cases := []struct {
		name string
		kind string
		spec map[string]interface{}
		want []string
	}{
		{
			name: "middleware_basicauth_and_forwardauth_tls",
			kind: "Middleware",
			spec: map[string]interface{}{
				"basicAuth":   map[string]interface{}{"secret": "htpasswd"},
				"forwardAuth": map[string]interface{}{"tls": map[string]interface{}{"certSecret": "fa-cert", "caSecret": "fa-ca"}},
			},
			want: []string{"fa-ca", "fa-cert", "htpasswd"},
		},
		{
			name: "middleware_forwardauth_url_only_no_secrets",
			kind: "Middleware",
			spec: map[string]interface{}{
				"forwardAuth": map[string]interface{}{"address": "http://10.1.2.10:9091/api/authz/forward-auth"},
			},
			want: nil,
		},
		{
			name: "middlewaretcp_never_has_secrets",
			kind: "MiddlewareTCP",
			spec: map[string]interface{}{"ipAllowList": map[string]interface{}{"sourceRange": []interface{}{"10.0.0.0/8"}}},
			want: nil,
		},
		{
			name: "tlsoption_clientauth_secretnames_list",
			kind: "TLSOption",
			spec: map[string]interface{}{
				"clientAuth": map[string]interface{}{"secretNames": []interface{}{"ca-a", "ca-b", "ca-a"}},
			},
			want: []string{"ca-a", "ca-b"},
		},
		{
			name: "serverstransport_cert_and_rootca_lists",
			kind: "ServersTransport",
			spec: map[string]interface{}{
				"certificatesSecrets": []interface{}{"client-cert"},
				"rootCAsSecrets":      []interface{}{"root-ca"},
			},
			want: []string{"client-cert", "root-ca"},
		},
		{
			name: "serverstransporttcp_nested_tls",
			kind: "ServersTransportTCP",
			spec: map[string]interface{}{
				"tls": map[string]interface{}{
					"certificatesSecrets": []interface{}{"tcp-cert"},
					"rootCAsSecrets":      []interface{}{"tcp-ca"},
				},
			},
			want: []string{"tcp-ca", "tcp-cert"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			obj := &unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": TraefikGroup + "/" + TraefikVersion,
				"kind":       c.kind,
				"metadata":   map[string]interface{}{"name": "x", "namespace": "src"},
				"spec":       c.spec,
			}}
			got := referencedSecrets(obj, refsForKind(c.kind))
			if len(got) == 0 && len(c.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("referencedSecrets = %v, want %v", got, c.want)
			}
		})
	}
}
