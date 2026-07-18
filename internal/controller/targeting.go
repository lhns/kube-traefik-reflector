package controller

import (
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/labels"
)

// NamespaceInfo is the minimal view of a Namespace the targeting logic needs.
type NamespaceInfo struct {
	Name        string
	Labels      map[string]string
	Terminating bool
}

// TargetingSpec holds the raw values of a source object's targeting annotations.
type TargetingSpec struct {
	Namespaces string // AnnNamespaces value: "a,b,c" or "*" or ""
	Selector   string // AnnNamespaceSelector value: a label selector or ""
}

// HasTargeting reports whether the source opts into reflection at all.
func (t TargetingSpec) HasTargeting() bool {
	return strings.TrimSpace(t.Namespaces) != "" || strings.TrimSpace(t.Selector) != ""
}

// parseList splits the AnnNamespaces value into explicit names and a wildcard flag.
func parseList(s string) (names []string, wildcard bool) {
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		switch {
		case p == "":
		case p == WildcardNamespaces:
			wildcard = true
		default:
			names = append(names, p)
		}
	}
	return names, wildcard
}

// resolveTargets computes the sorted set of namespaces a source in sourceNS should
// be mirrored into.
//
// Rules:
//   - "*" or a namespace-label selector expand over all live namespaces, but skip
//     the source's own namespace, terminating namespaces, the excluded set, and any
//     namespace carrying the opt-out label.
//   - An explicitly named namespace is honored even if it is in the excluded set or
//     carries the opt-out label (explicit intent wins) — but is still skipped when it
//     is the source's own namespace, is terminating, or does not exist.
func resolveTargets(sourceNS string, spec TargetingSpec, namespaces []NamespaceInfo, excluded map[string]bool) ([]string, error) {
	explicit, wildcard := parseList(spec.Namespaces)

	var selector labels.Selector
	if s := strings.TrimSpace(spec.Selector); s != "" {
		sel, err := labels.Parse(s)
		if err != nil {
			return nil, err
		}
		selector = sel
	}

	byName := make(map[string]NamespaceInfo, len(namespaces))
	for _, ns := range namespaces {
		byName[ns.Name] = ns
	}

	result := map[string]struct{}{}

	// Wildcard / selector expansion — subject to every exclusion.
	if wildcard || selector != nil {
		for _, ns := range namespaces {
			if ns.Name == sourceNS || ns.Terminating {
				continue
			}
			if excluded[ns.Name] || ns.Labels[LabelNamespaceExclude] == "true" {
				continue
			}
			if wildcard || selector.Matches(labels.Set(ns.Labels)) {
				result[ns.Name] = struct{}{}
			}
		}
	}

	// Explicit names — override the excluded set and opt-out label; only self,
	// terminating, or absent namespaces are skipped.
	for _, name := range explicit {
		if name == sourceNS {
			continue
		}
		ns, ok := byName[name]
		if !ok || ns.Terminating {
			continue
		}
		result[name] = struct{}{}
	}

	if len(result) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(result))
	for n := range result {
		out = append(out, n)
	}
	sort.Strings(out)
	return out, nil
}
