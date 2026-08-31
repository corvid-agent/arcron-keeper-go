package chain

import (
	"context"
	"testing"
	"time"

	"github.com/corvid-agent/arcron-keeper-go/internal/upkeep"
)

func TestCheckGenesisRefuseMainNet(t *testing.T) {
	err := CheckGenesis("mainnet-v1.0", "wGHE2Pwdvd7S12BL5FaOP20EGYesN73ktiC1qzkkit8=")
	if err == nil {
		t.Fatal("must refuse MainNet genesis")
	}
	if err := CheckGenesis(TestNetGenesisID, TestNetGenesisHashB64); err != nil {
		t.Fatal(err)
	}
}

func TestLiveBox19Target(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := Connect(ctx, DefaultAlgod, DefaultApp)
	if err != nil {
		t.Fatalf("connect TestNet: %v", err)
	}
	u, err := c.LoadUpkeep(ctx, 19)
	if err != nil {
		t.Fatal(err)
	}
	if u.Target != 769891902 {
		t.Fatalf("box 19 target=%d want 769891902", u.Target)
	}
	if u.Skip() {
		t.Fatal("box 19 must not be skipped")
	}
	raw, err := c.GetBox(ctx, upkeep.BoxKey(19))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < upkeep.HeadSize {
		t.Fatalf("short live box %d", len(raw))
	}
}

func TestLiveBox110PlodTarget(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := Connect(ctx, DefaultAlgod, DefaultApp)
	if err != nil {
		t.Fatalf("connect TestNet: %v", err)
	}
	u, err := c.LoadUpkeep(ctx, DefaultPlodUpkeep)
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != DefaultPlodUpkeep {
		t.Fatalf("box %d id mismatch", u.ID)
	}
	if u.Target != DefaultPlod {
		t.Fatalf("box %d target=%d want %d (Plod)", DefaultPlodUpkeep, u.Target, DefaultPlod)
	}
	if u.Skip() {
		t.Fatal("box 110 must not be skipped")
	}
	if u.Forbidden() {
		t.Fatal("box 110 must not be forbidden")
	}
	raw, err := c.GetBox(ctx, upkeep.BoxKey(DefaultPlodUpkeep))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < upkeep.HeadSize {
		t.Fatalf("short live plod box %d", len(raw))
	}
}

func TestDueUpkeepsSkips81(t *testing.T) {
	listed := []upkeep.Upkeep{
		{ID: 81, Next: 1, Fee: 4000, Balance: 8000},
		{ID: 19, Target: DefaultPulse, Next: 1, Fee: 4000, Balance: 8000},
		{ID: DefaultPlodUpkeep, Target: DefaultPlod, Next: 99_999_999, Fee: 4000, Balance: 8000},
		{ID: 87, Target: 770082145, Next: 1, Fee: 4000, Balance: 8000},
	}
	due := DueUpkeeps(listed, 100)
	if len(due) != 2 {
		t.Fatalf("due=%v want 19 and 87 (81 skipped from listing; 110 not due)", dueIDs(due))
	}
	if due[0].ID != 19 || due[1].ID != 87 {
		t.Fatalf("due ids %v", dueIDs(due))
	}
	for _, u := range due {
		if u.Skip() {
			t.Fatalf("due list leaked skipped upkeep %d", u.ID)
		}
	}
}

func dueIDs(listed []upkeep.Upkeep) []uint64 {
	ids := make([]uint64, len(listed))
	for i, u := range listed {
		ids[i] = u.ID
	}
	return ids
}
