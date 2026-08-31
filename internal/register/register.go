// Package register builds the Arcron register group: two payments (box MBR
// and escrow funding) plus the ABI app call. It never submits.
package register

import (
	"encoding/binary"
	"fmt"

	"github.com/algorand/go-algorand-sdk/v2/abi"
	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/algorand/go-algorand-sdk/v2/types"

	"github.com/corvid-agent/arcron-keeper-go/internal/upkeep"
)

// Signature is the keeper register method, matching Keeper.arc56.json.
const Signature = "register(pay,pay,uint64,byte[][],uint64,uint64,uint64,uint64,uint64,uint64)uint64"

// PulseTickSignature is the demo target's hook; the default call_args[0].
const PulseTickSignature = "tick()uint64"

// GroupSize is two payments plus the app call.
const GroupSize = 3

// BOX_MBR_FIXED mirrors the contract: 2,500 + 400 * (9-byte name + 130-byte head).
const BoxMBRFixed uint64 = 2500 + 400*139

// CATCH_UP / SKIP_AHEAD match the contract.
const CatchUp uint64 = 0
const SkipAhead uint64 = 1

const MinUpkeepFee uint64 = 4000
const MinInterval uint64 = 10

// Params is one unsigned register group.
type Params struct {
	AppID      uint64
	Sender     types.Address
	AppAddress types.Address
	Suggested  types.SuggestedParams
	NextID     uint64
	Target     uint64
	CallArgs   [][]byte
	Interval   uint64
	Fee        uint64
	Funding    uint64
	Policy     uint64
	FeeCap     uint64
	FeeAsset   uint64
	AssetFee   uint64
}

// Group is the three unsigned transactions, in order.
type Group struct {
	Txns     []types.Transaction
	BoxMBR   uint64
	Funding  uint64
	MinFee   uint64
	NextID   uint64
	BoxKey   []byte
	Selector []byte
}

// Method returns the ABI method.
func Method() (abi.Method, error) {
	return abi.MethodFromSignature(Signature)
}

// Selector is the 4-byte register selector.
func Selector() ([]byte, error) {
	m, err := Method()
	if err != nil {
		return nil, err
	}
	return m.GetSelector(), nil
}

// PulseTickSelector is the 4-byte tick()uint64 selector used as call_args[0]
// when pointing at Pulse.
func PulseTickSelector() ([]byte, error) {
	m, err := abi.MethodFromSignature(PulseTickSignature)
	if err != nil {
		return nil, err
	}
	return m.GetSelector(), nil
}

// EncodeCallArgs is the ARC-4 byte[][] tail the contract stores: uint16
// count, uint16 offset per argument (relative to the end of the count), then
// each argument's uint16 length and bytes.
func EncodeCallArgs(args [][]byte) []byte {
	n := len(args)
	header := 2 + 2*n
	total := header
	for _, a := range args {
		total += 2 + len(a)
	}
	out := make([]byte, total)
	binary.BigEndian.PutUint16(out[0:2], uint16(n))
	pos := header
	for i, a := range args {
		binary.BigEndian.PutUint16(out[2+2*i:], uint16(pos-2))
		binary.BigEndian.PutUint16(out[pos:pos+2], uint16(len(a)))
		copy(out[pos+2:], a)
		pos += 2 + len(a)
	}
	return out
}

// BoxMBR is what the MBR payment must cover.
func BoxMBR(callArgs [][]byte) uint64 {
	return BoxMBRFixed + 400*uint64(len(EncodeCallArgs(callArgs)))
}

func minFee(sp types.SuggestedParams) uint64 {
	min := uint64(sp.MinFee)
	if min == 0 {
		min = uint64(sp.Fee)
	}
	if min == 0 {
		min = 1000
	}
	return min
}

// Build constructs the unsigned 3-txn group. It does not sign or send.
func Build(p Params) (Group, error) {
	if p.Interval < MinInterval {
		return Group{}, fmt.Errorf("interval %d below minimum %d", p.Interval, MinInterval)
	}
	if p.Fee < MinUpkeepFee {
		return Group{}, fmt.Errorf("fee %d below minimum %d", p.Fee, MinUpkeepFee)
	}
	if len(p.CallArgs) == 0 || len(p.CallArgs) > 3 {
		return Group{}, fmt.Errorf("call_args count %d out of bounds (1..3)", len(p.CallArgs))
	}
	if p.Funding < p.Fee {
		return Group{}, fmt.Errorf("funding %d must cover at least one execution at fee %d", p.Funding, p.Fee)
	}
	method, err := Method()
	if err != nil {
		return Group{}, err
	}
	sel, err := Selector()
	if err != nil {
		return Group{}, err
	}
	mbr := BoxMBR(p.CallArgs)
	boxKey := upkeep.BoxKey(p.NextID)
	fee := minFee(p.Suggested)

	spPay := p.Suggested
	spPay.FlatFee = true
	spPay.Fee = types.MicroAlgos(fee)

	mbrPay, err := transaction.MakePaymentTxn(
		p.Sender.String(),
		p.AppAddress.String(),
		mbr,
		[]byte("arcron:mbr"),
		"",
		spPay,
	)
	if err != nil {
		return Group{}, err
	}
	fundPay, err := transaction.MakePaymentTxn(
		p.Sender.String(),
		p.AppAddress.String(),
		p.Funding,
		[]byte("arcron:funding"),
		"",
		spPay,
	)
	if err != nil {
		return Group{}, err
	}

	empty := transaction.EmptyTransactionSigner{}
	atc := transaction.AtomicTransactionComposer{}
	err = atc.AddMethodCall(transaction.AddMethodCallParams{
		AppID:  p.AppID,
		Method: method,
		MethodArgs: []interface{}{
			transaction.TransactionWithSigner{Txn: mbrPay, Signer: empty},
			transaction.TransactionWithSigner{Txn: fundPay, Signer: empty},
			p.Target,
			p.CallArgs,
			p.Interval,
			p.Fee,
			p.Policy,
			p.FeeCap,
			p.FeeAsset,
			p.AssetFee,
		},
		Sender:          p.Sender,
		SuggestedParams: spPay,
		Signer:          empty,
		BoxReferences: []types.AppBoxReference{{
			AppID: p.AppID,
			Name:  boxKey,
		}},
	})
	if err != nil {
		return Group{}, fmt.Errorf("compose register: %w", err)
	}
	built, err := atc.BuildGroup()
	if err != nil {
		return Group{}, err
	}
	if len(built) != GroupSize {
		return Group{}, fmt.Errorf("register group size %d, want %d", len(built), GroupSize)
	}
	txns := make([]types.Transaction, len(built))
	for i, tws := range built {
		txns[i] = tws.Txn
	}
	return Group{
		Txns:     txns,
		BoxMBR:   mbr,
		Funding:  p.Funding,
		MinFee:   fee,
		NextID:   p.NextID,
		BoxKey:   boxKey,
		Selector: sel,
	}, nil
}

// DefaultCallArgs is the Pulse tick()uint64 selector as a one-element list.
func DefaultCallArgs() ([][]byte, error) {
	sel, err := PulseTickSelector()
	if err != nil {
		return nil, err
	}
	return [][]byte{sel}, nil
}

// EphemeralSender is a throwaway address used only to fill Sender on an
// unsigned group. The secret is discarded; nothing is printed.
func EphemeralSender() types.Address {
	acct := crypto.GenerateAccount()
	return acct.Address
}
