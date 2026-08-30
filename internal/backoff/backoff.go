// Package backoff persists exponential wait after a target fails to accept
// the inner call. Lost races do not back off.
//
// Wait is 1, 2, 4, then 8 of the upkeep's own intervals, capped at 1286
// rounds (~one hour on TestNet) so a daily upkeep is not dark for a week.
package backoff

import (
	"encoding/json"
	"os"
	"sync"
)

// CapRounds is the absolute ceiling on a backoff wait (~1 hour of TestNet).
const CapRounds uint64 = 1286

// File is a JSON map of upkeep id → state, safe to share across --once runs.
type File struct {
	path string
	mu   sync.Mutex
	byID map[uint64]entry
}

type entry struct {
	Failures       uint64 `json:"failures"`
	SkipUntilRound uint64 `json:"skip_until_round"`
}

// Open loads an existing file or starts empty. path may be created on Save.
func Open(path string) (*File, error) {
	f := &File{path: path, byID: map[uint64]entry{}}
	if path == "" {
		return f, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return f, nil
		}
		return nil, err
	}
	if len(raw) == 0 {
		return f, nil
	}
	if err := json.Unmarshal(raw, &f.byID); err != nil {
		return nil, err
	}
	if f.byID == nil {
		f.byID = map[uint64]entry{}
	}
	return f, nil
}

// Blocked reports whether this upkeep should be left alone at lastRound.
func (f *File) Blocked(id, lastRound uint64) bool {
	if f == nil {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.byID[id]
	if !ok {
		return false
	}
	return lastRound < e.SkipUntilRound
}

// Fail records a target failure. interval is the upkeep's interval_rounds.
func (f *File) Fail(id, lastRound, interval uint64) error {
	if f == nil {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	e := f.byID[id]
	e.Failures++
	e.SkipUntilRound = lastRound + WaitRounds(e.Failures, interval)
	f.byID[id] = e
	return f.saveLocked()
}

// Success clears backoff after a confirmed execute.
func (f *File) Success(id uint64) error {
	if f == nil {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.byID, id)
	return f.saveLocked()
}

func (f *File) saveLocked() error {
	if f.path == "" {
		return nil
	}
	raw, err := json.MarshalIndent(f.byID, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(f.path, append(raw, '\n'), 0o600)
}

// WaitRounds is 1,2,4,8 × interval, never above CapRounds.
func WaitRounds(failures, interval uint64) uint64 {
	if failures == 0 {
		return 0
	}
	shift := failures - 1
	if shift > 3 {
		shift = 3
	}
	mult := uint64(1) << shift
	wait := interval * mult
	if interval == 0 || wait/mult != interval || wait > CapRounds {
		return CapRounds
	}
	return wait
}
