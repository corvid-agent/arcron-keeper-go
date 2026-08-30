package upkeep

import (
	"encoding/binary"
	"testing"
)

func TestBoxKeyRoundTrip(t *testing.T) {
	key := BoxKey(19)
	if len(key) != 9 || key[0] != 'u' {
		t.Fatalf("key=%x", key)
	}
	id, ok := ParseBoxKey(key)
	if !ok || id != 19 {
		t.Fatalf("parse %x -> %d ok=%v", key, id, ok)
	}
	if _, ok := ParseBoxKey([]byte("nope")); ok {
		t.Fatal("expected reject")
	}
}

func TestDecodeRejectsShortBuffer(t *testing.T) {
	_, err := Decode(1, make([]byte, 129))
	if err == nil {
		t.Fatal("expected short-buffer reject")
	}
}

func TestDecodeRejectsTailOffsetNot130(t *testing.T) {
	raw := make([]byte, 130)
	binary.BigEndian.PutUint16(raw[40:42], 128)
	_, err := Decode(1, raw)
	if err == nil {
		t.Fatal("expected tail-offset reject")
	}
}

func TestDecodeHeadAndDue(t *testing.T) {
	raw := make([]byte, 140)
	binary.BigEndian.PutUint64(raw[32:40], 769891902)
	binary.BigEndian.PutUint16(raw[40:42], 130)
	binary.BigEndian.PutUint64(raw[42:50], 10)
	binary.BigEndian.PutUint64(raw[50:58], 100)
	binary.BigEndian.PutUint64(raw[58:66], 4000)
	binary.BigEndian.PutUint64(raw[66:74], 8000)
	u, err := Decode(7, raw)
	if err != nil {
		t.Fatal(err)
	}
	if u.Target != 769891902 || u.Fee != 4000 || u.Balance != 8000 {
		t.Fatalf("%+v", u)
	}
	if !u.Due(100) || u.Due(99) {
		t.Fatalf("due logic: last=100 next=100 due=%v; last=99 due=%v", u.Due(100), u.Due(99))
	}
	starved := u
	starved.Balance = 3999
	if starved.Due(1000) {
		t.Fatal("starved upkeep must not be due")
	}
}

func TestSkip81(t *testing.T) {
	if !(Upkeep{ID: 81}).Skip() {
		t.Fatal("must skip 81")
	}
	if (Upkeep{ID: 82}).Skip() {
		t.Fatal("must not skip 82")
	}
}
