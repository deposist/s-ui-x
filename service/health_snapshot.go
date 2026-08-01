package service

import (
	"sort"
	"sync"
	"time"
)

// OutboundHealthSnapshot is the bounded operator-readable health view for one outbound.
type OutboundHealthSnapshot struct {
	Tag       string `json:"tag"`
	Status    string `json:"status"`
	DelayMs   uint16 `json:"delayMs,omitempty"`
	Error     string `json:"error,omitempty"`
	CheckedAt int64  `json:"checkedAt"`
}

var (
	outboundHealthMu         sync.RWMutex
	outboundHealthMap        = make(map[string]OutboundHealthSnapshot)
	outboundHealthMaxAge     = 10 * time.Minute
	outboundHealthMaxEntries = 1024
)

func SetOutboundHealth(tag string, ok bool, delayMs uint16, errMsg string) {
	RecordOutboundHealth(tag, ok, delayMs, errMsg, time.Now())
}

func RecordOutboundHealth(tag string, ok bool, delayMs uint16, errMsg string, checkedAt time.Time) OutboundHealthSnapshot {
	status := "healthy"
	if !ok {
		status = "down"
	}
	snapshot := OutboundHealthSnapshot{Tag: tag, Status: status, DelayMs: delayMs, Error: truncateHealthError(errMsg), CheckedAt: checkedAt.Unix()}
	outboundHealthMu.Lock()
	outboundHealthMap[tag] = snapshot
	pruneOutboundHealthLocked(checkedAt.Unix())
	outboundHealthMu.Unlock()
	return snapshot
}

func OutboundHealthSnapshotFor(tag string) (OutboundHealthSnapshot, bool) {
	cutoff := time.Now().Add(-outboundHealthMaxAge).Unix()
	outboundHealthMu.Lock()
	defer outboundHealthMu.Unlock()
	snapshot, ok := outboundHealthMap[tag]
	if !ok || snapshot.CheckedAt < cutoff {
		delete(outboundHealthMap, tag)
		return OutboundHealthSnapshot{}, false
	}
	return snapshot, true
}

func AllOutboundHealthSnapshots() map[string]OutboundHealthSnapshot {
	outboundHealthMu.Lock()
	defer outboundHealthMu.Unlock()
	pruneOutboundHealthLocked(time.Now().Unix())
	result := make(map[string]OutboundHealthSnapshot, len(outboundHealthMap))
	for tag, snapshot := range outboundHealthMap {
		result[tag] = snapshot
	}
	return result
}

func PruneOutboundHealth(keep []string) {
	keepSet := make(map[string]struct{}, len(keep))
	for _, tag := range keep {
		keepSet[tag] = struct{}{}
	}
	outboundHealthMu.Lock()
	for tag := range outboundHealthMap {
		if _, ok := keepSet[tag]; !ok {
			delete(outboundHealthMap, tag)
		}
	}
	outboundHealthMu.Unlock()
}

func ResetHealthSnapshots() {
	outboundHealthMu.Lock()
	outboundHealthMap = make(map[string]OutboundHealthSnapshot)
	outboundHealthMu.Unlock()
}

func pruneOutboundHealthLocked(now int64) {
	cutoff := now - int64(outboundHealthMaxAge/time.Second)
	for tag, snapshot := range outboundHealthMap {
		if snapshot.CheckedAt < cutoff {
			delete(outboundHealthMap, tag)
		}
	}
	if len(outboundHealthMap) <= outboundHealthMaxEntries {
		return
	}
	type entry struct {
		tag string
		at  int64
	}
	entries := make([]entry, 0, len(outboundHealthMap))
	for tag, snapshot := range outboundHealthMap {
		entries = append(entries, entry{tag: tag, at: snapshot.CheckedAt})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].at == entries[j].at {
			return entries[i].tag < entries[j].tag
		}
		return entries[i].at < entries[j].at
	})
	for i := 0; i < len(entries)-outboundHealthMaxEntries; i++ {
		delete(outboundHealthMap, entries[i].tag)
	}
}

func truncateHealthError(value string) string {
	const maxBytes = 200
	if len(value) <= maxBytes {
		return value
	}
	limit := maxBytes - 3
	for limit > 0 && (value[limit]&0xc0) == 0x80 {
		limit--
	}
	return value[:limit] + "..."
}
