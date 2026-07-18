package controller

import (
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// referencedSecrets returns the distinct, sorted set of Secret names referenced
// by obj according to refs. Missing fields are ignored.
func referencedSecrets(obj *unstructured.Unstructured, refs []SecretRef) []string {
	seen := map[string]struct{}{}
	add := func(name string) {
		if name != "" {
			seen[name] = struct{}{}
		}
	}
	for _, r := range refs {
		if r.List {
			vals, found, err := unstructured.NestedStringSlice(obj.Object, r.Path...)
			if err != nil || !found {
				continue
			}
			for _, v := range vals {
				add(v)
			}
			continue
		}
		v, found, err := unstructured.NestedString(obj.Object, r.Path...)
		if err != nil || !found {
			continue
		}
		add(v)
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
