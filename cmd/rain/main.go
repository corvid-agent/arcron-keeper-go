// Command rain lists RainRec boxes on TestNet hub 770130162. No key.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

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

func run() error {
	algodURL := flag.String("algod", chain.DefaultAlgod, "TestNet algod URL")
	hub := flag.Uint64("hub", rain.DefaultHub, "Rain hub app id")
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
	out := struct {
		Hub       uint64    `json:"hub"`
		LastRound uint64    `json:"last_round"`
		NextID    uint64    `json:"next_rain_id"`
		Rains     []rainRow `json:"rains"`
	}{Hub: *hub, LastRound: last, NextID: nextID}
	for id := uint64(1); id <= nextID; id++ {
		raw, err := c.GetBox(ctx, rain.BoxKey(id))
		if err != nil {
			return fmt.Errorf("rain %d: %w", id, err)
		}
		rec, err := rain.Decode(id, raw)
		if err != nil {
			return err
		}
		out.Rains = append(out.Rains, rainRow{
			ID: rec.ID, Label: rec.Label, Creator: types.Address(rec.Creator).String(),
			Mode: rain.ModeName(rec.Mode), Drip: rec.Drip, Pot: rec.Pot, Tickets: rec.Tickets,
			PrizeLocked: rec.PrizeLocked, CommitRound: rec.CommitRound, Status: rec.Status(last),
		})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
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
