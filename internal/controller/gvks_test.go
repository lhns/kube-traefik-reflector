package controller

import "testing"

func TestReflectedTypesTable(t *testing.T) {
	wantKinds := map[string]bool{
		"Middleware":          true,
		"MiddlewareTCP":       true,
		"TLSOption":           true,
		"ServersTransport":    true,
		"ServersTransportTCP": true,
	}

	seen := map[string]bool{}
	for _, rt := range ReflectedTypes {
		if rt.GVK.Group != TraefikGroup || rt.GVK.Version != TraefikVersion {
			t.Errorf("%s: gvk = %v, want group %s version %s", rt.GVK.Kind, rt.GVK, TraefikGroup, TraefikVersion)
		}
		if seen[rt.GVK.Kind] {
			t.Errorf("duplicate kind in table: %s", rt.GVK.Kind)
		}
		seen[rt.GVK.Kind] = true

		// Every secret-ref path must start at spec and be non-empty.
		for _, r := range rt.SecretRefs {
			if len(r.Path) < 2 || r.Path[0] != "spec" {
				t.Errorf("%s: secret ref path %v should start at spec", rt.GVK.Kind, r.Path)
			}
		}
	}

	for k := range wantKinds {
		if !seen[k] {
			t.Errorf("missing reflected kind: %s", k)
		}
	}
	if seen["IngressRoute"] || seen["IngressRouteTCP"] || seen["TraefikService"] {
		t.Error("routers / TraefikService must NOT be in the reflected table")
	}
}
