// Package upkeep decodes Arcron Upkeep boxes and decides whether one is due.
//
// The on-chain record is an ARC-4 head/tail struct. This package only reads
// the 130-byte head: that is enough to know who to call, when, and whether
// the escrow covers the base fee. v1 ignores fee escalation.
package upkeep

import (
	"encoding/binary"
	"fmt"
)

// HeadSize is the ARC-4 head length. The uint16 at [40:42] must equal this.
const HeadSize = 130

// SkippedID is never executed by this keeper. It is an agent-owned upkeep on
// the first-party CorvidLabs TestNet demo.
const SkippedID = 81

// Upkeep is the decoded 130-byte box head.
type Upkeep struct {
	ID       uint64
	Creator  [32]byte
	Target   uint64
	Interval uint64
	Next     uint64
	Fee      uint64
	Balance  uint64
	Times    uint64
	Policy   uint64
	Cap      uint64
	Last     uint64
	FeeAsset uint64
	AssetFee uint64
	AssetBal uint64
}

// BoxKey is the on-chain box name: 'u' || uint64 big-endian id.
func BoxKey(id uint64) []byte {
	key := make([]byte, 9)
	key[0] = 'u'
	binary.BigEndian.PutUint64(key[1:], id)
	return key
}

// ParseBoxKey extracts the upkeep id from a box name. Non-upkeep boxes
// (wrong prefix or length) return ok=false.
func ParseBoxKey(name []byte) (id uint64, ok bool) {
	if len(name) != 9 || name[0] != 'u' {
		return 0, false
	}
	return binary.BigEndian.Uint64(name[1:]), true
}

// Decode parses the 130-byte head. It rejects a short buffer and a tail
// offset that is not 130, rather than reading on into an older encoding.
func Decode(id uint64, raw []byte) (Upkeep, error) {
	if len(raw) < HeadSize {
		return Upkeep{}, fmt.Errorf("upkeep %d: short box (%d bytes, need %d)", id, len(raw), HeadSize)
	}
	offset := binary.BigEndian.Uint16(raw[40:42])
	if offset != HeadSize {
		return Upkeep{}, fmt.Errorf("upkeep %d: tail offset %d, want %d", id, offset, HeadSize)
	}
	u := Upkeep{ID: id}
	copy(u.Creator[:], raw[0:32])
	u.Target = binary.BigEndian.Uint64(raw[32:40])
	u.Interval = binary.BigEndian.Uint64(raw[42:50])
	u.Next = binary.BigEndian.Uint64(raw[50:58])
	u.Fee = binary.BigEndian.Uint64(raw[58:66])
	u.Balance = binary.BigEndian.Uint64(raw[66:74])
	u.Times = binary.BigEndian.Uint64(raw[74:82])
	u.Policy = binary.BigEndian.Uint64(raw[82:90])
	u.Cap = binary.BigEndian.Uint64(raw[90:98])
	u.Last = binary.BigEndian.Uint64(raw[98:106])
	u.FeeAsset = binary.BigEndian.Uint64(raw[106:114])
	u.AssetFee = binary.BigEndian.Uint64(raw[114:122])
	u.AssetBal = binary.BigEndian.Uint64(raw[122:130])
	return u, nil
}

// Due reports whether lastRound has reached next_execution_round and the
// escrow covers the base fee. Escalation (fee_cap) is ignored in v1.
func (u Upkeep) Due(lastRound uint64) bool {
	return lastRound >= u.Next && u.Balance >= u.Fee
}

// Skip reports whether this keeper must not touch the upkeep.
func (u Upkeep) Skip() bool {
	return u.ID == SkippedID
}
