// Package rain decodes the 224-byte RainRec box on TestNet hub 770130162.
package rain

import (
	"encoding/binary"
	"fmt"
)

const RecSize = 224
const SeedWindow = 800

// DefaultHub is the immutable TestNet Rain hub.
const DefaultHub uint64 = 770130162

type Rec struct {
	ID            uint64
	Creator       [32]byte
	GateCreator   [32]byte
	Label         string
	PrizeAsset    uint64
	Drip          uint64
	Interval      uint64
	LastRainRound uint64
	Pot           uint64
	Tickets       uint64
	DrawID        uint64
	Cumulative    uint64
	Mode          uint64
	WaveCap       uint64
	WaveCount     uint64
	LastShare     uint64
	LastWaveID    uint64
	WaveUnclaimed uint64
	CommitRound   uint64
	PrizeLocked   uint64
}

func BoxKey(id uint64) []byte {
	key := make([]byte, 9)
	key[0] = 'r'
	binary.BigEndian.PutUint64(key[1:], id)
	return key
}

func ParseBoxKey(name []byte) (uint64, bool) {
	if len(name) != 9 || name[0] != 'r' {
		return 0, false
	}
	return binary.BigEndian.Uint64(name[1:]), true
}

func Decode(id uint64, raw []byte) (Rec, error) {
	if len(raw) < RecSize {
		return Rec{}, fmt.Errorf("rain %d: short RainRec len=%d want>=%d", id, len(raw), RecSize)
	}
	r := Rec{ID: id}
	copy(r.Creator[:], raw[0:32])
	copy(r.GateCreator[:], raw[32:64])
	r.Label = labelOf(raw[64:96])
	r.PrizeAsset = binary.BigEndian.Uint64(raw[96:104])
	r.Drip = binary.BigEndian.Uint64(raw[104:112])
	r.Interval = binary.BigEndian.Uint64(raw[112:120])
	r.LastRainRound = binary.BigEndian.Uint64(raw[120:128])
	r.Pot = binary.BigEndian.Uint64(raw[128:136])
	r.Tickets = binary.BigEndian.Uint64(raw[136:144])
	r.DrawID = binary.BigEndian.Uint64(raw[144:152])
	r.Cumulative = binary.BigEndian.Uint64(raw[152:160])
	r.Mode = binary.BigEndian.Uint64(raw[160:168])
	r.WaveCap = binary.BigEndian.Uint64(raw[168:176])
	r.WaveCount = binary.BigEndian.Uint64(raw[176:184])
	r.LastShare = binary.BigEndian.Uint64(raw[184:192])
	r.LastWaveID = binary.BigEndian.Uint64(raw[192:200])
	r.WaveUnclaimed = binary.BigEndian.Uint64(raw[200:208])
	r.CommitRound = binary.BigEndian.Uint64(raw[208:216])
	r.PrizeLocked = binary.BigEndian.Uint64(raw[216:224])
	return r, nil
}

func labelOf(chunk []byte) string {
	n := len(chunk)
	for i, b := range chunk {
		if b == 0 {
			n = i
			break
		}
	}
	return string(chunk[:n])
}

func ModeName(mode uint64) string {
	switch mode {
	case 0:
		return "SPLIT"
	case 1:
		return "ONE"
	case 2:
		return "WAVE"
	default:
		return fmt.Sprintf("%d", mode)
	}
}

// Status at lastRound. Only ONE with prize_locked uses the 800-round window.
func (r Rec) Status(lastRound uint64) string {
	if r.Mode == 1 && r.PrizeLocked > 0 {
		if lastRound <= r.CommitRound {
			return "drawn-waiting-resolve"
		}
		if lastRound <= r.CommitRound+SeedWindow {
			return "resolve-window remaining"
		}
		return "abandonable"
	}
	return "open"
}
