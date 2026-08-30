// Command keeper is an independent Go bot that executes due Arcron upkeeps
// on Algorand TestNet. It refuses any other genesis.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/mnemonic"

	"github.com/corvid-agent/arcron-keeper-go/internal/backoff"
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
	loadDotEnv(".env")
	algodURL := flag.String("algod", envOr("ALGOD_URL", chain.DefaultAlgod), "TestNet algod URL")
	appID := flag.Uint64("app-id", envU64("KEEPER_APP_ID", chain.DefaultApp), "Arcron keeper app id")
	dryRun := flag.Bool("dry-run", false, "print due upkeeps and sign nothing")
	once := flag.Bool("once", false, "scan once (and execute unless --dry-run)")
	backoffFile := flag.String("backoff-file", envOr("BACKOFF_FILE", "backoff.json"), "persisted target-failure backoff")
	poll := flag.Duration("poll", 4*time.Second, "loop interval when neither --once nor --dry-run")
	only := flag.Uint64("upkeep", 0, "if set, only consider this upkeep id")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	c, err := chain.Connect(ctx, *algodURL, *appID)
	if err != nil {
		return err
	}

	bf, err := backoff.Open(*backoffFile)
	if err != nil {
		return err
	}

	if *dryRun {
		return scan(ctx, c, bf, true, nil, *only)
	}

	mn := os.Getenv("KEEPER_MNEMONIC")
	if mn == "" {
		return fmt.Errorf("KEEPER_MNEMONIC is required unless --dry-run")
	}
	sk, err := mnemonic.ToPrivateKey(mn)
	if err != nil {
		return fmt.Errorf("mnemonic: %w", err)
	}
	acct, err := crypto.AccountFromPrivateKey(sk)
	if err != nil {
		return err
	}
	b := &execute.Builder{Client: c, Account: acct, Backoff: bf}
	log.Printf("keeper %s app %d genesis TestNet", acct.Address.String(), c.AppID)

	if *once {
		return scan(ctx, c, bf, false, b, *only)
	}
	for {
		if err := scan(ctx, c, bf, false, b, *only); err != nil {
			log.Printf("scan: %v", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(*poll):
		}
	}
}

func scan(ctx context.Context, c *chain.Client, bf *backoff.File, dry bool, b *execute.Builder, only uint64) error {
	last, err := c.LastRound(ctx)
	if err != nil {
		return err
	}
	listed, errs := c.ListUpkeeps(ctx)
	for _, e := range errs {
		log.Printf("decode: %v", e)
	}
	if only != 0 {
		filtered := listed[:0]
		for _, u := range listed {
			if u.ID == only {
				filtered = append(filtered, u)
			}
		}
		listed = filtered
	}
	due := chain.DueUpkeeps(listed, last)
	sort.Slice(due, func(i, j int) bool { return due[i].ID < due[j].ID })

	var actionable []upkeep.Upkeep
	for _, u := range due {
		if bf.Blocked(u.ID, last) {
			log.Printf("backoff upkeep %d until later", u.ID)
			continue
		}
		actionable = append(actionable, u)
	}

	log.Printf("round %d: %d upkeeps, %d due (skip %d always, %d backed off)",
		last, len(listed), len(due), countSkip(listed), len(due)-len(actionable))

	for _, u := range actionable {
		if dry {
			log.Printf("due upkeep %d target %d next %d fee %d balance %d (dry-run, not signed)",
				u.ID, u.Target, u.Next, u.Fee, u.Balance)
			continue
		}
		res, err := b.Call(ctx, u)
		if err != nil {
			log.Printf("execute(%d) failed: %v", u.ID, err)
			continue
		}
		if res.Skipped {
			log.Printf("skipped upkeep %d", u.ID)
			continue
		}
		log.Printf("executed upkeep %d tx %s round %d", u.ID, res.TxID, res.Round)
	}
	return nil
}

func countSkip(listed []upkeep.Upkeep) int {
	n := 0
	for _, u := range listed {
		if u.Skip() {
			n++
		}
	}
	return n
}

func loadDotEnv(path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = trimQuotes(val)
		if key != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envU64(key string, fallback uint64) uint64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n uint64
	if _, err := fmt.Sscan(v, &n); err != nil {
		return fallback
	}
	return n
}

func trimQuotes(val string) string {
	if len(val) >= 2 {
		if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
			return val[1 : len(val)-1]
		}
	}
	return val
}
