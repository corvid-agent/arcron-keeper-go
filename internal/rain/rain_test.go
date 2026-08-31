package rain

import (
	"testing"
)

func TestBoxKey(t *testing.T) {
	key := BoxKey(3)
	if len(key) != 9 || key[0] != 'r' {
		t.Fatalf("key=%x", key)
	}
	id, ok := ParseBoxKey(key)
	if !ok || id != 3 {
		t.Fatalf("parse %d %v", id, ok)
	}
}

func TestDecodeRejectsShort(t *testing.T) {
	if _, err := Decode(1, make([]byte, 223)); err == nil {
		t.Fatal("expected short reject")
	}
}

func TestStatusWindow(t *testing.T) {
	r := Rec{Mode: 1, PrizeLocked: 50000, CommitRound: 100}
	if r.Status(100) != "drawn-waiting-resolve" {
		t.Fatal(r.Status(100))
	}
	if r.Status(101) != "resolve-window remaining" {
		t.Fatal(r.Status(101))
	}
	if r.Status(100+SeedWindow+1) != "abandonable" {
		t.Fatal(r.Status(100 + SeedWindow + 1))
	}
	if (Rec{}).Status(9) != "open" {
		t.Fatal("split is open")
	}
}
