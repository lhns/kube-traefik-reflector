package controller

import (
	"reflect"
	"testing"
)

func nsList(specs ...NamespaceInfo) []NamespaceInfo { return specs }

func ns(name string, opts ...func(*NamespaceInfo)) NamespaceInfo {
	n := NamespaceInfo{Name: name, Labels: map[string]string{}}
	for _, o := range opts {
		o(&n)
	}
	return n
}

func withLabel(k, v string) func(*NamespaceInfo) {
	return func(n *NamespaceInfo) { n.Labels[k] = v }
}
func terminating() func(*NamespaceInfo) {
	return func(n *NamespaceInfo) { n.Terminating = true }
}

func TestResolveTargets(t *testing.T) {
	all := nsList(
		ns("authelia"), // the source namespace in most cases
		ns("duckdb"),
		ns("dashy"),
		ns("web", withLabel("team", "web")),
		ns("data", withLabel("team", "data")),
		ns("kube-system"),
		ns("locked", withLabel(LabelNamespaceExclude, "true")),
		ns("dying", terminating()),
	)
	excluded := map[string]bool{"kube-system": true, "kube-public": true, "kube-node-lease": true}

	cases := []struct {
		name     string
		sourceNS string
		spec     TargetingSpec
		want     []string
		wantErr  bool
	}{
		{
			name:     "explicit_list",
			sourceNS: "authelia",
			spec:     TargetingSpec{Namespaces: "duckdb,dashy"},
			want:     []string{"dashy", "duckdb"},
		},
		{
			name:     "wildcard_excludes_system_self_terminating_optout",
			sourceNS: "authelia",
			spec:     TargetingSpec{Namespaces: "*"},
			want:     []string{"dashy", "data", "duckdb", "web"},
		},
		{
			name:     "selector_matches_labels",
			sourceNS: "authelia",
			spec:     TargetingSpec{Selector: "team=web"},
			want:     []string{"web"},
		},
		{
			name:     "explicit_overrides_excluded_set",
			sourceNS: "authelia",
			spec:     TargetingSpec{Namespaces: "kube-system,duckdb"},
			want:     []string{"duckdb", "kube-system"},
		},
		{
			name:     "explicit_overrides_optout_label",
			sourceNS: "authelia",
			spec:     TargetingSpec{Namespaces: "locked"},
			want:     []string{"locked"},
		},
		{
			name:     "self_skipped_even_if_named",
			sourceNS: "duckdb",
			spec:     TargetingSpec{Namespaces: "duckdb,dashy"},
			want:     []string{"dashy"},
		},
		{
			name:     "terminating_skipped_even_if_named",
			sourceNS: "authelia",
			spec:     TargetingSpec{Namespaces: "dying,dashy"},
			want:     []string{"dashy"},
		},
		{
			name:     "absent_namespace_skipped",
			sourceNS: "authelia",
			spec:     TargetingSpec{Namespaces: "ghost,dashy"},
			want:     []string{"dashy"},
		},
		{
			name:     "list_and_selector_union",
			sourceNS: "authelia",
			spec:     TargetingSpec{Namespaces: "duckdb", Selector: "team=data"},
			want:     []string{"data", "duckdb"},
		},
		{
			name:     "no_targeting_yields_nothing",
			sourceNS: "authelia",
			spec:     TargetingSpec{},
			want:     nil,
		},
		{
			name:     "invalid_selector_errors",
			sourceNS: "authelia",
			spec:     TargetingSpec{Selector: "team=@bad"},
			wantErr:  true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := resolveTargets(c.sourceNS, c.spec, all, excluded)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (result %v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("resolveTargets = %v, want %v", got, c.want)
			}
		})
	}
}

func TestHasTargeting(t *testing.T) {
	cases := []struct {
		name string
		spec TargetingSpec
		want bool
	}{
		{"empty", TargetingSpec{}, false},
		{"whitespace_only", TargetingSpec{Namespaces: "  "}, false},
		{"namespaces", TargetingSpec{Namespaces: "a"}, true},
		{"selector", TargetingSpec{Selector: "x=y"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.spec.HasTargeting(); got != c.want {
				t.Errorf("HasTargeting() = %v, want %v", got, c.want)
			}
		})
	}
}
