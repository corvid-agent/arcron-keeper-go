// Command listen lists Arcron upkeep boxes on TestNet algod and writes
// docs/due.json. It does not sign. No mnemonic. Not an execute.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/corvid-agent/arcron-keeper-go/internal/chain"
	"github.com/corvid-agent/arcron-keeper-go/internal/upkeep"
)

func main() {
	log.SetFlags(0)
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	algodURL := flag.String("algod", chain.DefaultAlgod, "TestNet algod URL")
	appID := flag.Uint64("app-id", chain.DefaultApp, "Arcron keeper app id")
	out := flag.String("out", "docs/due.json", "output path for due.json")
	flag.Parse()

	ctx := context.Background()
	c, err := chain.Connect(ctx, *algodURL, *appID)
	if err != nil {
		return err
	}
	lastRound, err := c.LastRound(ctx)
	if err != nil {
		return err
	}
	listed, errs := c.ListUpkeeps(ctx)
	for _, e := range errs {
		log.Printf("decode: %v", e)
	}
	sort.Slice(listed, func(i, j int) bool { return listed[i].ID < listed[j].ID })

	skipped := make([]uint64, 0)
	due := make([]dueEntry, 0)
	for _, u := range listed {
		if u.Skip() {
			skipped = append(skipped, u.ID)
			continue
		}
		if u.Due(lastRound) {
			due = append(due, toDueEntry(u))
		}
	}

	payload := report{
		GeneratedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		Algod:       *algodURL,
		App:         c.AppID,
		Genesis:     chain.TestNetGenesisID,
		LastRound:   lastRound,
		Listed:      len(listed),
		Skipped:     skipped,
		DueCount:    len(due),
		Due:         due,
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(*out, raw, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s last_round=%d listed=%d due=%d skipped=%v\n",
		*out, lastRound, len(listed), len(due), skipped)
	return nil
}

type report struct {
	GeneratedAt string     `json:"generated_at"`
	Algod       string     `json:"algod"`
	App         uint64     `json:"app"`
	Genesis     string     `json:"genesis"`
	LastRound   uint64     `json:"last_round"`
	Listed      int        `json:"listed"`
	Skipped     []uint64   `json:"skipped"`
	DueCount    int        `json:"due_count"`
	Due         []dueEntry `json:"due"`
}

type dueEntry struct {
	ID       uint64 `json:"id"`
	Target   uint64 `json:"target"`
	Interval uint64 `json:"interval"`
	Next     uint64 `json:"next"`
	Fee      uint64 `json:"fee"`
	Balance  uint64 `json:"balance"`
	Times    uint64 `json:"times"`
}

func toDueEntry(u upkeep.Upkeep) dueEntry {
	return dueEntry{
		ID:       u.ID,
		Target:   u.Target,
		Interval: u.Interval,
		Next:     u.Next,
		Fee:      u.Fee,
		Balance:  u.Balance,
		Times:    u.Times,
	}
}
