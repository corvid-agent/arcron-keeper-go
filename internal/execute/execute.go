// Package execute builds and (optionally) submits Arcron execute(uint64) calls.
package execute

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/algorand/go-algorand-sdk/v2/abi"
	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/algorand/go-algorand-sdk/v2/types"

	"github.com/corvid-agent/arcron-keeper-go/internal/backoff"
	"github.com/corvid-agent/arcron-keeper-go/internal/chain"
	"github.com/corvid-agent/arcron-keeper-go/internal/upkeep"
)

// ExtraFeeMicroAlgos covers the two inner transactions (call + payment)
// via fee pooling. Outer min fee is separate.
const ExtraFeeMicroAlgos uint64 = 2000

const executeSig = "execute(uint64)uint64"

// Result is a confirmed execute, or empty TxID if nothing was sent.
type Result struct {
	UpkeepID uint64
	TxID     string
	Round    uint64
	Skipped  bool
	DryRun   bool
}

// Builder holds the bits needed to form an execute call.
type Builder struct {
	Client  *chain.Client
	Account crypto.Account
	Backoff *backoff.File
}

// AppArgs is the ARC-4 payload for execute(upkeep_id).
func AppArgs(id uint64) ([][]byte, error) {
	method, err := abi.MethodFromSignature(executeSig)
	if err != nil {
		return nil, err
	}
	arg := make([]byte, 8)
	binary.BigEndian.PutUint64(arg, id)
	return [][]byte{method.GetSelector(), arg}, nil
}

// Unsigned builds the NoOp app call: box u||id, foreign app = target,
// extra fee 2000 µALGO. It does not sign.
func (b *Builder) Unsigned(ctx context.Context, u upkeep.Upkeep) (types.Transaction, error) {
	if u.Skip() {
		return types.Transaction{}, fmt.Errorf("refusing to execute skipped upkeep %d", u.ID)
	}
	args, err := AppArgs(u.ID)
	if err != nil {
		return types.Transaction{}, err
	}
	sp, err := b.Client.Algod.SuggestedParams().Do(ctx)
	if err != nil {
		return types.Transaction{}, err
	}
	sp.FlatFee = true
	min := uint64(sp.MinFee)
	if min == 0 {
		min = uint64(sp.Fee)
	}
	if min == 0 {
		min = 1000
	}
	sp.Fee = types.MicroAlgos(min + ExtraFeeMicroAlgos)

	boxes := []types.AppBoxReference{{
		AppID: b.Client.AppID,
		Name:  upkeep.BoxKey(u.ID),
	}}
	foreign := []uint64{u.Target}
	tx, err := transaction.MakeApplicationNoOpTxWithBoxes(
		b.Client.AppID,
		args,
		nil,
		foreign,
		nil,
		boxes,
		sp,
		b.Account.Address,
		nil,
		types.Digest{},
		[32]byte{},
		types.Address{},
	)
	if err != nil {
		return types.Transaction{}, err
	}
	return tx, nil
}

// Call signs and submits execute(upkeep_id). Skip 81. On a lost race the
// backoff file is left alone; on a target failure it is updated.
func (b *Builder) Call(ctx context.Context, u upkeep.Upkeep) (Result, error) {
	if u.Skip() {
		return Result{UpkeepID: u.ID, Skipped: true}, nil
	}
	tx, err := b.Unsigned(ctx, u)
	if err != nil {
		return Result{}, err
	}
	_, stx, err := crypto.SignTransaction(b.Account.PrivateKey, tx)
	if err != nil {
		return Result{}, err
	}
	txid, err := b.Client.Algod.SendRawTransaction(stx).Do(ctx)
	if err != nil {
		b.handleFailure(ctx, u, err)
		return Result{}, fmt.Errorf("broadcast execute(%d): %w", u.ID, err)
	}
	conf, err := wait(ctx, b.Client.Algod, txid)
	if err != nil {
		b.handleFailure(ctx, u, err)
		return Result{}, fmt.Errorf("confirm execute(%d) %s: %w", u.ID, txid, err)
	}
	if b.Backoff != nil {
		_ = b.Backoff.Success(u.ID)
	}
	return Result{UpkeepID: u.ID, TxID: txid, Round: conf}, nil
}

func wait(ctx context.Context, c *algod.Client, txid string) (uint64, error) {
	resp, err := transaction.WaitForConfirmation(c, txid, 8, ctx)
	if err != nil {
		return 0, err
	}
	return resp.ConfirmedRound, nil
}

func (b *Builder) handleFailure(ctx context.Context, u upkeep.Upkeep, err error) {
	if b.Backoff == nil || err == nil {
		return
	}
	if isRace(ctx, b.Client, u, err) {
		return
	}
	last, lerr := b.Client.LastRound(ctx)
	if lerr != nil {
		last = 0
	}
	_ = b.Backoff.Fail(u.ID, last, u.Interval)
}

// isRace is true when another keeper already took this slot: the box moved,
// or the node rejected a schedule check rather than an inner call.
func isRace(ctx context.Context, c *chain.Client, before upkeep.Upkeep, err error) bool {
	msg := strings.ToLower(err.Error())
	fresh, lerr := c.LoadUpkeep(ctx, before.ID)
	if lerr == nil {
		if fresh.Next != before.Next || fresh.Times != before.Times {
			return true
		}
	}
	if strings.Contains(msg, "inner") || strings.Contains(msg, "innertransaction") {
		return false
	}
	if strings.Contains(msg, "logic eval error") && !strings.Contains(msg, "inner") {
		// schedule assert, typically a lost race that has not yet landed
		return true
	}
	return false
}

// SelectorHex is the 4-byte execute selector, for tests and docs.
func SelectorHex() string {
	method, err := abi.MethodFromSignature(executeSig)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", method.GetSelector())
}

// FeeOnTxn returns the fee the unsigned txn will pay.
func FeeOnTxn(tx types.Transaction) uint64 {
	return uint64(tx.Fee)
}

// SameSelector is a compile-time-ish check that our encoding is the ABI one.
func SameSelector() bool {
	want, _ := abi.MethodFromSignature(executeSig)
	return bytes.Equal(want.GetSelector(), []byte{0x5b, 0x49, 0xcc, 0x5c})
}
