// Command decode fetches one Arcron upkeep box from TestNet algod and prints
// the 130-byte head as JSON. It refuses any non-TestNet genesis.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/algorand/go-algorand-sdk/v2/types"

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
	id := flag.Uint64("id", 0, "upkeep box id")
	algodURL := flag.String("algod", chain.DefaultAlgod, "TestNet algod URL")
	appID := flag.Uint64("app-id", chain.DefaultApp, "Arcron keeper app id")
	flag.Parse()
	if *id == 0 {
		return fmt.Errorf("--id is required")
	}

	ctx := context.Background()
	c, err := chain.Connect(ctx, *algodURL, *appID)
	if err != nil {
		return err
	}
	u, err := c.LoadUpkeep(ctx, *id)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(asHeadJSON(u))
}

type headJSON struct {
	ID       uint64 `json:"id"`
	Creator  string `json:"creator"`
	Target   uint64 `json:"target"`
	Interval uint64 `json:"interval"`
	Next     uint64 `json:"next"`
	Fee      uint64 `json:"fee"`
	Balance  uint64 `json:"balance"`
	Times    uint64 `json:"times"`
	Policy   uint64 `json:"policy"`
	Cap      uint64 `json:"cap"`
	Last     uint64 `json:"last"`
	FeeAsset uint64 `json:"fee_asset"`
	AssetFee uint64 `json:"asset_fee"`
	AssetBal uint64 `json:"asset_bal"`
}

func asHeadJSON(u upkeep.Upkeep) headJSON {
	return headJSON{
		ID:       u.ID,
		Creator:  types.Address(u.Creator).String(),
		Target:   u.Target,
		Interval: u.Interval,
		Next:     u.Next,
		Fee:      u.Fee,
		Balance:  u.Balance,
		Times:    u.Times,
		Policy:   u.Policy,
		Cap:      u.Cap,
		Last:     u.Last,
		FeeAsset: u.FeeAsset,
		AssetFee: u.AssetFee,
		AssetBal: u.AssetBal,
	}
}
