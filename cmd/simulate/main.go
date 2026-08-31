// Command simulate asks TestNet algod to evaluate execute(uint64) against due
// boxes with empty signatures. It never signs and never sends. Skips 81.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/algorand/go-algorand-sdk/v2/types"

	"github.com/corvid-agent/arcron-keeper-go/internal/chain"
	"github.com/corvid-agent/arcron-keeper-go/internal/execute"
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
	only := flag.Uint64("upkeep", 0, "if set, only this id (still refuses 81 and 87)")
	flag.Parse()

	ctx := context.Background()
	c, err := chain.Connect(ctx, *algodURL, *appID)
	if err != nil {
		return err
	}
	last, err := c.LastRound(ctx)
	if err != nil {
		return err
	}
	listed, errs := c.ListUpkeeps(ctx)
	for _, e := range errs {
		log.Printf("decode: %v", e)
	}
	sort.Slice(listed, func(i, j int) bool { return listed[i].ID < listed[j].ID })

	var due []upkeep.Upkeep
	for _, u := range listed {
		if *only != 0 && u.ID != *only {
			continue
		}
		if u.Skip() {
			continue
		}
		if *only == 0 && !u.Due(last) {
			continue
		}
		due = append(due, u)
	}

	results := make([]execute.SimResult, 0, len(due))
	for _, u := range due {
		sender := types.Address(u.Creator)
		res, err := execute.Simulate(ctx, c, sender, u)
		if err != nil {
			res = execute.SimResult{
				UpkeepID:       u.ID,
				Target:         u.Target,
				Selector:       execute.SelectorHex(),
				FailureMessage: err.Error(),
				Sent:           false,
			}
		}
		results = append(results, res)
		if res.Skipped {
			log.Printf("skip simulate upkeep %d (%s)", u.ID, res.SkipReason)
			continue
		}
		if res.WouldSucceed {
			log.Printf("simulate execute(%d) would succeed round %d budget %d (not sent)",
				u.ID, res.LastRound, res.AppBudget)
		} else {
			log.Printf("simulate execute(%d) would fail: %s (not sent)", u.ID, res.FailureMessage)
		}
	}

	payload := map[string]any{
		"dry_run":    true,
		"sent":       false,
		"algod":      *algodURL,
		"genesis":    chain.TestNetGenesisID,
		"app":        c.AppID,
		"last_round": last,
		"selector":   execute.SelectorHex(),
		"due_count":  len(due),
		"results":    results,
		"note":       "no-key algod simulate of execute; empty signatures; nothing broadcast. 81 skipped. Not a live execute.",
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "simulate: last_round=%d due=%d sent=false\n", last, len(due))
	return nil
}
