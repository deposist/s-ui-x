package cronjob

import (
	"context"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/service"
)

func newTestFailoverJob(now func() time.Time, probe func(string) bool, active *string, switches *[]string) *FailoverJob {
	return &FailoverJob{
		states:       map[string]*failoverGroupState{},
		now:          now,
		probe:        func(_ context.Context, tag, _ string) bool { return probe(tag) },
		activeMember: func(string) (string, bool) { return *active, true },
		switchMember: func(_, target string) error { *switches = append(*switches, target); *active = target; return nil },
	}
}

// Drives the manager through outage and recovery: fast failover to the backup,
// sticky while the primary is unconfirmed, then hysteresis-gated failback.
func TestFailoverJobFailoverThenFailback(t *testing.T) {
	cur := time.Unix(1000, 0)
	active := "a"
	var switches []string
	health := map[string]bool{"a": true, "b": true}

	j := newTestFailoverJob(func() time.Time { return cur }, func(tag string) bool { return health[tag] }, &active, &switches)
	group := service.FailoverGroupConfig{Tag: "g", Members: []string{"a", "b"}, ProbeTarget: "x", Interval: 30 * time.Second, Hysteresis: 2, Enabled: true}

	tick := func() {
		j.runGroup(nil, group, "")
		cur = cur.Add(group.Interval)
	}

	tick()              // a up, b up; on a -> sticky
	health["a"] = false // primary fails
	tick()              // a down -> failover to b
	health["a"] = true  // primary recovering
	tick()              // a up1 (unconfirmed) -> stay on b
	tick()              // a up2 (confirmed)   -> failback to a

	if len(switches) != 2 || switches[0] != "b" || switches[1] != "a" {
		t.Fatalf("switch sequence = %v, want [b a]", switches)
	}
	if active != "a" {
		t.Fatalf("final active = %q, want a", active)
	}
}

// When every member is down and a direct fallback exists, route through direct.
func TestFailoverJobAllDownToDirect(t *testing.T) {
	cur := time.Unix(1000, 0)
	active := "a"
	var switches []string
	j := newTestFailoverJob(func() time.Time { return cur }, func(string) bool { return false }, &active, &switches)
	group := service.FailoverGroupConfig{Tag: "g", Members: []string{"a", "b"}, ProbeTarget: "x", Interval: 30 * time.Second, Hysteresis: 2, Enabled: true}

	j.runGroup(nil, group, "direct")

	if len(switches) != 1 || switches[0] != "direct" {
		t.Fatalf("all-down switches = %v, want [direct]", switches)
	}
}

// A disabled group must never be probed or switched.
func TestFailoverJobDisabledGroupIsSkipped(t *testing.T) {
	cur := time.Unix(1000, 0)
	active := "a"
	var switches []string
	j := newTestFailoverJob(func() time.Time { return cur }, func(string) bool { return false }, &active, &switches)
	group := service.FailoverGroupConfig{Tag: "g", Members: []string{"a", "b"}, ProbeTarget: "x", Interval: 30 * time.Second, Hysteresis: 2, Enabled: false}

	j.runGroup(nil, group, "direct")

	if len(switches) != 0 {
		t.Fatalf("disabled group must not switch, got %v", switches)
	}
}
