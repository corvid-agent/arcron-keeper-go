// Command rain lists RainRec boxes on TestNet hub 770130162 and writes
// docs/rain.json. No key. Not a send.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/types"
	"github.com/corvid-agent/arcron-keeper-go/internal/chain"
	"github.com/corvid-agent/arcron-keeper-go/internal/rain"
)

func main() {
	log.SetFlags(0)
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

type rainRow struct {
	ID          uint64 `json:"id"`
	Label       string `json:"label"`
	Creator     string `json:"creator"`
	Mode        string `json:"mode"`
	Drip        uint64 `json:"drip"`
	Pot         uint64 `json:"pot"`
	Tickets     uint64 `json:"tickets"`
	PrizeLocked uint64 `json:"prize_locked"`
	CommitRound uint64 `json:"commit_round"`
	Status      string `json:"status"`
}

type report struct {
	GeneratedAt string    `json:"generated_at"`
	Algod       string    `json:"algod"`
	Hub         uint64    `json:"hub"`
	Genesis     string    `json:"genesis"`
	LastRound   uint64    `json:"last_round"`
	NextID      uint64    `json:"next_rain_id"`
	Rains       []rainRow `json:"rains"`
}

func run() error {
	algodURL := flag.String("algod", chain.DefaultAlgod, "TestNet algod URL")
	hub := flag.Uint64("hub", rain.DefaultHub, "Rain hub app id")
	out := flag.String("out", "docs/rain.json", "output path for rain.json")
	flag.Parse()
	ctx := context.Background()
	c, err := chain.Connect(ctx, *algodURL, *hub)
	if err != nil {
		return err
	}
	last, err := c.LastRound(ctx)
	if err != nil {
		return err
	}
	nextID, err := globalUint(ctx, c, "next_rain_id")
	if err != nil {
		return err
	}
	payload := report{
		GeneratedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		Algod:       *algodURL,
		Hub:         *hub,
		Genesis:     chain.TestNetGenesisID,
		LastRound:   last,
		NextID:      nextID,
		Rains:       []rainRow{},
	}
	for id := uint64(1); id <= nextID; id++ {
		raw, err := c.GetBox(ctx, rain.BoxKey(id))
		if err != nil {
			return fmt.Errorf("rain %d: %w", id, err)
		}
		rec, err := rain.Decode(id, raw)
		if err != nil {
			return err
		}
		payload.Rains = append(payload.Rains, rainRow{
			ID: rec.ID, Label: rec.Label, Creator: types.Address(rec.Creator).String(),
			Mode: rain.ModeName(rec.Mode), Drip: rec.Drip, Pot: rec.Pot, Tickets: rec.Tickets,
			PrizeLocked: rec.PrizeLocked, CommitRound: rec.CommitRound, Status: rec.Status(last),
		})
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(*out, body, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s last_round=%d rains=%d\n", *out, last, len(payload.Rains))
	return nil
}

func globalUint(ctx context.Context, c *chain.Client, want string) (uint64, error) {
	app, err := c.Algod.GetApplicationByID(c.AppID).Do(ctx)
	if err != nil {
		return 0, err
	}
	for _, kv := range app.Params.GlobalState {
		key := kv.Key
		if raw, err := base64.StdEncoding.DecodeString(key); err == nil {
			key = string(raw)
		}
		if key == want {
			return kv.Value.Uint, nil
		}
	}
	return 0, fmt.Errorf("app %d: %s missing", c.AppID, want)
}
