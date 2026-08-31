package execute

import (
	"encoding/binary"
	"testing"

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
