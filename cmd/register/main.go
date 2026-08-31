// Command register prints the unsigned Arcron register group (MBR payment,
// funding payment, app call) against TestNet. It never signs and never sends.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/types"

	"github.com/corvid-agent/arcron-keeper-go/internal/chain"
	"github.com/corvid-agent/arcron-keeper-go/internal/register"
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
	target := flag.Uint64("target", chain.DefaultPulse, "target app id (default Pulse)")
	interval := flag.Uint64("interval", 100, "interval_rounds")
	fee := flag.Uint64("fee", 10000, "fee_per_execution µALGO")
	funding := flag.Uint64("funding", 40000, "escrow funding µALGO")
	policy := flag.Uint64("policy", register.SkipAhead, "0=CATCH_UP 1=SKIP_AHEAD")
	senderStr := flag.String("sender", "", "sender address (default: ephemeral throwaway, secret discarded)")
	flag.Parse()

	ctx := context.Background()
	c, err := chain.Connect(ctx, *algodURL, *appID)
	if err != nil {
		return err
	}
	sp, err := c.Algod.SuggestedParams().Do(ctx)
	if err != nil {
		return err
	}
	nextID, err := c.NextUpkeepID(ctx)
	if err != nil {
		return err
	}
	callArgs, err := register.DefaultCallArgs()
	if err != nil {
		return err
	}

	var sender types.Address
	senderNote := "ephemeral throwaway; secret discarded, never printed"
	if *senderStr != "" {
		sender, err = types.DecodeAddress(*senderStr)
		if err != nil {
			return fmt.Errorf("--sender: %w", err)
		}
		senderNote = "supplied --sender; no key loaded"
	} else {
		sender = crypto.GenerateAccount().Address
	}

	g, err := register.Build(register.Params{
		AppID:      c.AppID,
		Sender:     sender,
		AppAddress: c.AppAddress(),
		Suggested:  sp,
		NextID:     nextID,
		Target:     *target,
		CallArgs:   callArgs,
		Interval:   *interval,
		Fee:        *fee,
		Funding:    *funding,
		Policy:     *policy,
	})
	if err != nil {
		return err
	}

	type txnJSON struct {
		Index      int      `json:"index"`
		Type       string   `json:"type"`
		Role       string   `json:"role"`
		Sender     string   `json:"sender"`
		Receiver   string   `json:"receiver,omitempty"`
		Amount     uint64   `json:"amount,omitempty"`
		Fee        uint64   `json:"fee"`
		Note       string   `json:"note,omitempty"`
		AppID      uint64   `json:"app_id,omitempty"`
		OnComplete string   `json:"on_complete,omitempty"`
		Boxes      []string `json:"boxes,omitempty"`
		AppArgs    []string `json:"app_args_hex,omitempty"`
		Group      string   `json:"group"`
	}
	roles := []string{"mbr_payment", "funding_payment", "app_call"}
	outTxns := make([]txnJSON, len(g.Txns))
	for i, tx := range g.Txns {
		row := txnJSON{
			Index:  i,
			Type:   string(tx.Type),
			Role:   roles[i],
			Sender: tx.Sender.String(),
			Fee:    uint64(tx.Fee),
			Group:  hex.EncodeToString(tx.Group[:]),
		}
		if tx.Type == types.PaymentTx {
			row.Receiver = tx.Receiver.String()
			row.Amount = uint64(tx.Amount)
			row.Note = string(tx.Note)
		}
		if tx.Type == types.ApplicationCallTx {
			row.AppID = uint64(tx.ApplicationID)
			row.OnComplete = "NoOp"
			for _, b := range tx.BoxReferences {
				row.Boxes = append(row.Boxes, hex.EncodeToString(b.Name))
			}
			for _, a := range tx.ApplicationArgs {
				row.AppArgs = append(row.AppArgs, hex.EncodeToString(a))
			}
		}
		outTxns[i] = row
	}

	payload := map[string]any{
		"dry_run":           true,
		"sent":              false,
		"signed":            false,
		"algod":             *algodURL,
		"genesis":           chain.TestNetGenesisID,
		"app":               c.AppID,
		"app_address":       c.AppAddress().String(),
		"sender":            sender.String(),
		"sender_note":       senderNote,
		"next_upkeep_id":    nextID,
		"box_key_hex":       hex.EncodeToString(g.BoxKey),
		"box_key_len":       len(g.BoxKey),
		"method":            register.Signature,
		"selector_hex":      hex.EncodeToString(g.Selector),
		"target":            *target,
		"call_args_hex":     []string{hex.EncodeToString(callArgs[0])},
		"interval_rounds":   *interval,
		"fee_per_execution": *fee,
		"policy":            *policy,
		"group_size":        len(g.Txns),
		"group":             outTxns,
		"cost_microalgo": map[string]uint64{
			"box_deposit":  g.BoxMBR,
			"escrow":       g.Funding,
			"network_fees": g.MinFee * uint64(register.GroupSize),
			"total":        g.BoxMBR + g.Funding + g.MinFee*uint64(register.GroupSize),
		},
		"note": "unsigned register group; not submitted. There is no --send.",
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "register dry-run: group_size=%d next_upkeep_id=%d sent=false\n", len(g.Txns), nextID)
	return nil
}
