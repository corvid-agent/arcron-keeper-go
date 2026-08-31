package chain

import (
	"context"
	"testing"
	"time"

	"github.com/corvid-agent/arcron-keeper-go/internal/rain"
)

func TestLiveRain3Hub(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := Connect(ctx, DefaultAlgod, rain.DefaultHub)
	if err != nil {
		t.Fatalf("connect rain hub: %v", err)
	}
	raw, err := c.GetBox(ctx, rain.BoxKey(3))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < rain.RecSize {
		t.Fatalf("short rain 3 box %d want>=%d", len(raw), rain.RecSize)
	}
	rec, err := rain.Decode(3, raw)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Mode != 1 {
		t.Fatalf("rain 3 mode=%d want 1 (ONE)", rec.Mode)
	}
	if rec.CommitRound == 0 {
		t.Fatal("rain 3 missing commit_round")
	}
}
