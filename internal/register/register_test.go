package register

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"testing"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/types"

	"github.com/corvid-agent/arcron-keeper-go/internal/chain"
	"github.com/corvid-agent/arcron-keeper-go/internal/upkeep"
)

func TestRegisterSelectorStable(t *testing.T) {
	sel, err := Selector()
	if err != nil {
		t.Fatal(err)
	}
	if len(sel) != 4 {
		t.Fatalf("selector len %d", len(sel))
	}
	// Pinned so a signature change is a failing test, not a silent group reshape.
	got := hex.EncodeToString(sel)
	if got != "3636cfc6" {
		t.Fatalf("register selector %s want 3636cfc6", got)
	}
}

func TestPulseTickSelectorFourBytes(t *testing.T) {
	sel, err := PulseTickSelector()
	if err != nil {
		t.Fatal(err)
	}
	if len(sel) != 4 {
		t.Fatalf("tick selector %x", sel)
	}
}

func TestEncodeCallArgsAndBoxMBR(t *testing.T) {
	sel, err := PulseTickSelector()
	if err != nil {
		t.Fatal(err)
	}
	encoded := EncodeCallArgs([][]byte{sel})
	if len(encoded) != 10 {
		t.Fatalf("encoded len %d want 10 (count+offset+len+4)", len(encoded))
	}
	if binary.BigEndian.Uint16(encoded[0:2]) != 1 {
		t.Fatalf("count %d", binary.BigEndian.Uint16(encoded[0:2]))
	}
	mbr := BoxMBR([][]byte{sel})
	want := BoxMBRFixed + 400*10
	if mbr != want {
		t.Fatalf("mbr %d want %d", mbr, want)
	}
}

func TestBoxKeyNineBytes(t *testing.T) {
	key := upkeep.BoxKey(111)
	if len(key) != 9 || key[0] != 'u' {
		t.Fatalf("key=%x", key)
	}
	if binary.BigEndian.Uint64(key[1:]) != 111 {
		t.Fatalf("id %x", key)
	}
}

func TestBuildGroupShapeNoSend(t *testing.T) {
	args, err := DefaultCallArgs()
	if err != nil {
		t.Fatal(err)
	}
	sender := crypto.GenerateAccount().Address
	appID := uint64(769891898)
	g, err := Build(Params{
		AppID:      appID,
		Sender:     sender,
		AppAddress: crypto.GetApplicationAddress(appID),
		Suggested: types.SuggestedParams{
			Fee:             1000,
			GenesisID:       "testnet-v1.0",
			GenesisHash:     mustHash(),
			FirstRoundValid: 1,
			LastRoundValid:  1001,
			MinFee:          1000,
		},
		NextID:   111,
		Target:   769891902,
		CallArgs: args,
		Interval: 100,
		Fee:      4000,
		Funding:  40000,
		Policy:   SkipAhead,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Txns) != GroupSize {
		t.Fatalf("group %d", len(g.Txns))
	}
	if g.Txns[0].Type != types.PaymentTx || g.Txns[1].Type != types.PaymentTx {
		t.Fatalf("want pay,pay got %s %s", g.Txns[0].Type, g.Txns[1].Type)
	}
	if g.Txns[2].Type != types.ApplicationCallTx {
		t.Fatalf("want appl got %s", g.Txns[2].Type)
	}
	if uint64(g.Txns[0].Amount) != g.BoxMBR {
		t.Fatalf("mbr amount %d want %d", g.Txns[0].Amount, g.BoxMBR)
	}
	if uint64(g.Txns[1].Amount) != 40000 {
		t.Fatalf("funding %d", g.Txns[1].Amount)
	}
	if string(g.Txns[0].Note) != "arcron:mbr" || string(g.Txns[1].Note) != "arcron:funding" {
		t.Fatalf("notes %q %q", g.Txns[0].Note, g.Txns[1].Note)
	}
	if len(g.BoxKey) != 9 || g.BoxKey[0] != 'u' {
		t.Fatalf("box key %x", g.BoxKey)
	}
	if g.Txns[2].ApplicationID != types.AppIndex(appID) {
		t.Fatalf("app %d", g.Txns[2].ApplicationID)
	}
	if len(g.Txns[2].ApplicationArgs) < 1 {
		t.Fatal("missing method selector")
	}
	if string(g.Txns[2].ApplicationArgs[0]) != string(g.Selector) {
		t.Fatalf("selector %x vs %x", g.Txns[2].ApplicationArgs[0], g.Selector)
	}
	// Group IDs assigned; nothing was (or can be) broadcast from this test.
	var zero types.Digest
	if g.Txns[0].Group == zero || g.Txns[0].Group != g.Txns[2].Group {
		t.Fatal("group id not assigned across the three txns")
	}
}

func TestBuildRejectsStarvedFunding(t *testing.T) {
	args, err := DefaultCallArgs()
	if err != nil {
		t.Fatal(err)
	}
	_, err = Build(Params{
		AppID:      1,
		Sender:     crypto.GenerateAccount().Address,
		AppAddress: crypto.GetApplicationAddress(1),
		Suggested:  types.SuggestedParams{Fee: 1000, MinFee: 1000, FirstRoundValid: 1, LastRoundValid: 2, GenesisID: "testnet-v1.0", GenesisHash: mustHash()},
		NextID:     1,
		Target:     2,
		CallArgs:   args,
		Interval:   10,
		Fee:        4000,
		Funding:    3999,
	})
	if err == nil {
		t.Fatal("must reject funding below one fee")
	}
}

func TestLiveRegisterDryRunUnsigned(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := chain.Connect(ctx, chain.DefaultAlgod, chain.DefaultApp)
	if err != nil {
		t.Fatalf("connect TestNet: %v", err)
	}
	nextID, err := c.NextUpkeepID(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if nextID == 0 {
		t.Fatal("next_upkeep_id is 0")
	}
	sp, err := c.Algod.SuggestedParams().Do(ctx)
	if err != nil {
		t.Fatal(err)
	}
	args, err := DefaultCallArgs()
	if err != nil {
		t.Fatal(err)
	}
	sender := EphemeralSender()
	g, err := Build(Params{
		AppID:      c.AppID,
		Sender:     sender,
		AppAddress: c.AppAddress(),
		Suggested:  sp,
		NextID:     nextID,
		Target:     chain.DefaultPulse,
		CallArgs:   args,
		Interval:   100,
		Fee:        10000,
		Funding:    40000,
		Policy:     SkipAhead,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Txns) != GroupSize {
		t.Fatalf("group %d", len(g.Txns))
	}
	if uint64(g.Txns[2].ApplicationID) != chain.DefaultApp {
		t.Fatalf("app %d", g.Txns[2].ApplicationID)
	}
	if binary.BigEndian.Uint64(g.BoxKey[1:]) != nextID {
		t.Fatalf("box key id %x want %d", g.BoxKey, nextID)
	}
	if string(g.Txns[0].Note) != "arcron:mbr" || string(g.Txns[1].Note) != "arcron:funding" {
		t.Fatalf("notes %q %q", g.Txns[0].Note, g.Txns[1].Note)
	}
	if g.Txns[0].GenesisID != chain.TestNetGenesisID {
		t.Fatalf("genesis %q", g.Txns[0].GenesisID)
	}
	if sp.FirstRoundValid == 0 {
		t.Fatal("live suggested params missing first valid round")
	}
	// Build returns types.Transaction, not SignedTxn. Nothing is broadcast.
	var zero types.Digest
	if g.Txns[0].Group == zero || g.Txns[0].Group != g.Txns[2].Group {
		t.Fatal("group id not assigned across the three txns")
	}
}

func mustHash() []byte {
	h := make([]byte, 32)
	h[0] = 1
	return h
}
