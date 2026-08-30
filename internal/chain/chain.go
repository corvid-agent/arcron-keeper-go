// Package chain talks to a TestNet-only algod. Any other genesis is refused.
package chain

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/algorand/go-algorand-sdk/v2/client/v2/algod"
	"github.com/corvid-agent/arcron-keeper-go/internal/upkeep"
)

// TestNetGenesisHashB64 is the unique identifier of Algorand TestNet.
const TestNetGenesisHashB64 = "SGO1GKSzyE7IEPItTxCByw9x8FmnrCDexi9/cOUJOiI="

// TestNetGenesisID is the human-readable TestNet id. Hash is the real check.
const TestNetGenesisID = "testnet-v1.0"

// DefaultAlgod is the public TestNet algod this keeper is built against.
const DefaultAlgod = "https://testnet-api.algonode.cloud"

// DefaultApp is the live Arcron keeper on TestNet (alpha-3).
const DefaultApp uint64 = 769891898

// Client is an algod handle that has already passed the TestNet genesis gate.
type Client struct {
	Algod   *algod.Client
	AppID   uint64
	Genesis string
}

// Connect dials algod and refuses any genesis that is not TestNet.
func Connect(ctx context.Context, url string, appID uint64) (*Client, error) {
	if url == "" {
		url = DefaultAlgod
	}
	if appID == 0 {
		appID = DefaultApp
	}
	algodClient, err := algod.MakeClient(url, "")
	if err != nil {
		return nil, err
	}
	ver, err := algodClient.Versions().Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("algod versions: %w", err)
	}
	hashB64 := base64.StdEncoding.EncodeToString(ver.GenesisHash)
	if err := CheckGenesis(ver.GenesisID, hashB64); err != nil {
		return nil, err
	}
	return &Client{Algod: algodClient, AppID: appID, Genesis: hashB64}, nil
}

// LastRound is the chain tip.
func (c *Client) LastRound(ctx context.Context) (uint64, error) {
	st, err := c.Algod.Status().Do(ctx)
	if err != nil {
		return 0, err
	}
	return st.LastRound, nil
}

// BoxNames lists box names on the keeper app.
func (c *Client) BoxNames(ctx context.Context) ([][]byte, error) {
	resp, err := c.Algod.GetApplicationBoxes(c.AppID).Do(ctx)
	if err != nil {
		return nil, err
	}
	names := make([][]byte, 0, len(resp.Boxes))
	for _, b := range resp.Boxes {
		names = append(names, b.Name)
	}
	return names, nil
}

// GetBox fetches one box by raw name.
func (c *Client) GetBox(ctx context.Context, name []byte) ([]byte, error) {
	resp, err := c.Algod.GetApplicationBoxByName(c.AppID, name).Do(ctx)
	if err != nil {
		return nil, err
	}
	return resp.Value, nil
}

// LoadUpkeep fetches and decodes one upkeep box.
func (c *Client) LoadUpkeep(ctx context.Context, id uint64) (upkeep.Upkeep, error) {
	raw, err := c.GetBox(ctx, upkeep.BoxKey(id))
	if err != nil {
		return upkeep.Upkeep{}, err
	}
	return upkeep.Decode(id, raw)
}

// ListUpkeeps walks the app's boxes and decodes every well-formed upkeep.
// Unknown box names and decode failures are skipped with an error collected.
func (c *Client) ListUpkeeps(ctx context.Context) ([]upkeep.Upkeep, []error) {
	names, err := c.BoxNames(ctx)
	if err != nil {
		return nil, []error{err}
	}
	var out []upkeep.Upkeep
	var errs []error
	for _, name := range names {
		id, ok := upkeep.ParseBoxKey(name)
		if !ok {
			continue
		}
		raw, err := c.GetBox(ctx, name)
		if err != nil {
			errs = append(errs, fmt.Errorf("box %d: %w", id, err))
			continue
		}
		u, err := upkeep.Decode(id, raw)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out = append(out, u)
	}
	return out, errs
}

// DueUpkeeps returns upkeeps that are due at lastRound, excluding 81.
func DueUpkeeps(listed []upkeep.Upkeep, lastRound uint64) []upkeep.Upkeep {
	var due []upkeep.Upkeep
	for _, u := range listed {
		if u.Skip() {
			continue
		}
		if u.Due(lastRound) {
			due = append(due, u)
		}
	}
	return due
}

// CheckGenesis refuses anything that is not Algorand TestNet.
func CheckGenesis(genesisID, genesisHashB64 string) error {
	if genesisHashB64 != TestNetGenesisHashB64 || genesisID != TestNetGenesisID {
		return fmt.Errorf("refusing non-TestNet genesis id=%q hash=%q (want id=%s hash=%s)",
			genesisID, genesisHashB64, TestNetGenesisID, TestNetGenesisHashB64)
	}
	return nil
}

// EncodeNameB64 is useful for debug logs.
func EncodeNameB64(name []byte) string {
	return "b64:" + base64.StdEncoding.EncodeToString(name)
}
