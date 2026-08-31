package execute

import (
	"encoding/binary"
	"testing"

	"github.com/algorand/go-algorand-sdk/v2/types"

	"github.com/corvid-agent/arcron-keeper-go/internal/upkeep"
)

func TestExecuteSelector(t *testing.T) {
	if !SameSelector() {
		t.Fatalf("selector %s != 5b49cc5c", SelectorHex())
	}
	args, err := AppArgs(19)
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 2 || len(args[0]) != 4 {
		t.Fatalf("%x", args)
	}
	if binary.BigEndian.Uint64(args[1]) != 19 {
		t.Fatalf("id arg %x", args[1])
	}
}

func TestCallRefusesSkipInUnsignedPath(t *testing.T) {
	b := &Builder{}
	_, err := b.Unsigned(nil, upkeep.Upkeep{ID: 81, Target: 1})
	if err == nil {
		t.Fatal("must refuse unsigned execute of 81")
	}
	_, err = b.Unsigned(nil, upkeep.Upkeep{ID: 87, Target: 770082145})
	if err == nil {
		t.Fatal("must refuse unsigned execute of 87")
	}
}

func TestSimulateSkips81WithoutAlgod(t *testing.T) {
	res, err := Simulate(nil, nil, types.Address{}, upkeep.Upkeep{ID: 81, Target: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Skipped {
		t.Fatal("simulate must skip 81 without talking to algod")
	}
	if res.Sent || res.WouldSucceed {
		t.Fatalf("skip must not look like a send or success: %+v", res)
	}
	if res.UpkeepID != 81 {
		t.Fatalf("upkeep id %d", res.UpkeepID)
	}
	if res.SkipReason == "" {
		t.Fatal("skip reason required")
	}
	if res.Selector != "5b49cc5c" {
		t.Fatalf("selector %s", res.Selector)
	}
}

func TestCallSkipsForbiddenWithoutBroadcast(t *testing.T) {
	b := &Builder{}
	res, err := b.Call(nil, upkeep.Upkeep{ID: 81, Target: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Skipped || res.TxID != "" {
		t.Fatalf("call 81 must skip without a txid: %+v", res)
	}
	res, err = b.Call(nil, upkeep.Upkeep{ID: 87, Target: 770082145})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Skipped || res.TxID != "" {
		t.Fatalf("call 87 must skip without a txid: %+v", res)
	}
}
