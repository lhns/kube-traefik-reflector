package chart

import (
	"os/exec"
	"strings"
	"testing"
)

const chartDir = "../../charts/kube-traefik-reflector"

func render(t *testing.T, set ...string) string {
	t.Helper()
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm not on PATH")
	}
	args := []string{"template", "r", chartDir}
	for _, s := range set {
		args = append(args, "--set", s)
	}
	out, err := exec.Command("helm", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("helm template: %v\n%s", err, out)
	}
	return string(out)
}

func TestLeaderElectionTimings(t *testing.T) {
	const prefix = "--leader-election-"
	timings := []string{
		"leaderElection.leaseDuration=60s",
		"leaderElection.renewDeadline=40s",
		"leaderElection.retryPeriod=5s",
	}

	t.Run("set values render flags", func(t *testing.T) {
		out := render(t, timings...)
		for _, want := range []string{
			prefix + "lease-duration=60s",
			prefix + "renew-deadline=40s",
			prefix + "retry-period=5s",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("missing %s", want)
			}
		}
	})
	t.Run("unset values render no flags", func(t *testing.T) {
		if out := render(t); strings.Contains(out, prefix) {
			t.Errorf("unexpected %s flag with defaults", prefix)
		}
	})
	t.Run("disabled leader election renders no flags", func(t *testing.T) {
		out := render(t, append(timings, "leaderElection.enabled=false")...)
		if strings.Contains(out, prefix) {
			t.Errorf("unexpected %s flag with leader election off", prefix)
		}
	})
}
