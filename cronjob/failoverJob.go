package cronjob

import (
	"context"
	"sync"
	"time"

	"github.com/deposist/s-ui-x/core"
	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/service"
)

// failoverProbeConcurrency bounds how many member probes run at once per cycle,
// matching DoctorService.outboundChecks so a many-member panel can't exhaust
// dialers.
const failoverProbeConcurrency = 4

type failoverGroupState struct {
	lastProbe time.Time
	health    map[string]service.MemberHealth
}

// FailoverJob is the in-process auto-failover manager. Every base tick it probes
// the due groups' members (reusing CheckOutbound), runs the pure selection
// decision, and switches the active member via the sing-box selector. It holds
// NO core/selector pointers between ticks (re-resolves each cycle) and only
// per-group rolling health under its mutex — so it survives core restarts.
type FailoverJob struct {
	service.ConfigService

	mu     sync.Mutex
	states map[string]*failoverGroupState

	// now is injectable for deterministic tests.
	now func() time.Time
	// probe is injectable for tests; defaults to the live CheckOutbound.
	probe func(ctx context.Context, tag, target string) bool
	// switchMember is injectable for tests; defaults to the live core.
	switchMember func(groupTag, memberTag string) error
	// activeMember is injectable for tests; defaults to the live core.
	activeMember func(groupTag string) (string, bool)
}

func NewFailoverJob() *FailoverJob {
	return &FailoverJob{
		states: make(map[string]*failoverGroupState),
		now:    time.Now,
	}
}

func (j *FailoverJob) Run() {
	db := database.GetDB()
	groups, err := service.LoadFailoverGroups(db)
	if err != nil {
		logger.Warning("failover: load groups: ", err)
		return
	}
	if len(groups) == 0 {
		return
	}
	coreInst := service.DefaultRuntime().Core()
	if coreInst == nil || !coreInst.IsRunning() {
		return
	}
	directTag := service.DirectFallbackTag(db)
	for _, group := range groups {
		j.runGroup(coreInst, group, directTag)
	}
}

func (j *FailoverJob) runGroup(coreInst *core.Core, group service.FailoverGroupConfig, directTag string) {
	if !group.Enabled || len(group.Members) == 0 {
		return
	}

	j.mu.Lock()
	st := j.states[group.Tag]
	if st == nil {
		st = &failoverGroupState{health: map[string]service.MemberHealth{}}
		j.states[group.Tag] = st
	}
	due := st.lastProbe.IsZero() || j.now().Sub(st.lastProbe) >= group.Interval
	j.mu.Unlock()
	if !due {
		return
	}

	results := j.probeMembers(group)

	j.mu.Lock()
	for _, member := range group.Members {
		h := st.health[member]
		if results[member] {
			h.ConsecutiveUp++
			h.ConsecutiveDown = 0
		} else {
			h.ConsecutiveDown++
			h.ConsecutiveUp = 0
		}
		st.health[member] = h
	}
	st.lastProbe = j.now()
	snapshot := make(map[string]service.MemberHealth, len(st.health))
	for k, v := range st.health {
		snapshot[k] = v
	}
	j.mu.Unlock()

	j.persistState(group, snapshot)

	current, _ := j.now0(coreInst, group.Tag)

	fallback := ""
	if directTag != "" && !memberListContains(group.Members, directTag) {
		fallback = directTag
	}

	decision := service.SelectFailoverMember(service.FailoverDecisionInput{
		Members:        group.Members,
		Health:         snapshot,
		Current:        current,
		Hysteresis:     group.Hysteresis,
		DirectFallback: fallback,
	})
	if !decision.ShouldSwitch {
		return
	}
	if err := j.switch0(coreInst, group.Tag, decision.Target); err != nil {
		logger.Warning("failover: switch ", group.Tag, " -> ", decision.Target, ": ", err)
		return
	}
	logger.Info("failover: group ", group.Tag, " -> ", decision.Target, " (", decision.Reason, ")")
}

func (j *FailoverJob) probeMembers(group service.FailoverGroupConfig) map[string]bool {
	results := make(map[string]bool, len(group.Members))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, failoverProbeConcurrency)
	ctx, cancel := context.WithTimeout(context.Background(), group.Interval)
	defer cancel()
	for _, member := range group.Members {
		wg.Add(1)
		go func(tag string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			var ok bool
			if j.probe != nil {
				ok = j.probe(ctx, tag, group.ProbeTarget)
			} else {
				ok = j.ConfigService.CheckOutboundWithContext(ctx, tag, group.ProbeTarget).OK
			}
			mu.Lock()
			results[tag] = ok
			mu.Unlock()
		}(member)
	}
	wg.Wait()
	return results
}

// now0 reads the group's active member, preferring the test hook.
func (j *FailoverJob) now0(coreInst *core.Core, groupTag string) (string, bool) {
	if j.activeMember != nil {
		return j.activeMember(groupTag)
	}
	return coreInst.GroupNow(groupTag)
}

// switch0 applies a member switch, preferring the test hook.
func (j *FailoverJob) switch0(coreInst *core.Core, groupTag, memberTag string) error {
	if j.switchMember != nil {
		return j.switchMember(groupTag, memberTag)
	}
	return coreInst.SelectGroupMember(groupTag, memberTag)
}

// persistState records the per-member health to the observability table. A nil
// database (unit tests) makes this a no-op.
func (j *FailoverJob) persistState(group service.FailoverGroupConfig, snapshot map[string]service.MemberHealth) {
	nowUnix := j.now().Unix()
	states := make([]service.FailoverMemberState, 0, len(group.Members))
	for _, m := range group.Members {
		h := snapshot[m]
		states = append(states, service.FailoverMemberState{
			GroupTag:    group.Tag,
			MemberTag:   m,
			Healthy:     h.ConsecutiveUp >= 1,
			ConsecUp:    h.ConsecutiveUp,
			ConsecDown:  h.ConsecutiveDown,
			LastProbeAt: nowUnix,
		})
	}
	_ = service.WriteFailoverMemberStates(database.GetDB(), states)
}

func memberListContains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
