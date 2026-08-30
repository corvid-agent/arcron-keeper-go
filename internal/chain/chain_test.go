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
