package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	accountv1 "github.com/openpak/account-api/openpak/account/v1"
	"openpak/np-common/coreclient"
	"openpak/np-common/fakecore"
	"openpak/np-common/identity"
	"openpak/np-common/ticket"
	"openpak/np-trophy/native"
)

func TestNativeTrophyUsesPersistentStoreAndIsolatedListener(t *testing.T) {
	dbURL := os.Getenv("NP_TROPHY_TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("NP_TROPHY_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := &Store{DB: pool}
	if err = store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, "TRUNCATE trophy_unlocks"); err != nil {
		t.Fatal(err)
	}
	const coreKey = "test-core-key-at-least-thirty-two-bytes"
	const account = "0b7c3f4e-1d2a-4c5b-9e8f-112233445566"
	fake := fakecore.New(coreKey)
	fake.AddAccount(fakecore.Account{ID: account, DisplayName: "Alice", Country: "US"})
	core, err := coreclient.Dial("passthrough:///core", coreKey, fake.Serve(t)...)
	if err != nil {
		t.Fatal(err)
	}
	defer core.Close()
	acct, err := core.GetAccount(ctx, account)
	if err != nil {
		t.Fatal(err)
	}
	link, err := identity.EnsureNPLink(ctx, core, acct)
	if err != nil {
		t.Fatal(err)
	}
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ticket.New(key, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := signer.Issue(ticket.Identity{AccountID: account, LinkID: link.LinkID, OnlineID: link.OnlineID, Country: "us", Service: "IV0001-NPXS01001_00"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	(&native.Service{Core: core, Tickets: signer, Trophies: nativeStore{store}}).Register(mux)
	server := httptest.NewServer(mux)
	defer server.Close()
	post := func(body string) int {
		resp, err := http.Post(server.URL+"/trophy/sec/sync_trophies", "text/xml", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body)
		return resp.StatusCode
	}
	if code := post(`<nptrophy/>`); code != 401 {
		t.Fatalf("missing ticket: %d", code)
	}
	body := `<nptrophy><ticket>` + base64.StdEncoding.EncodeToString(raw) + `</ticket><list><info npcommid="NPWR01234_00"><trophies><trophy id="1" type="0">2026-10-15T12:00:00Z</trophy></trophies></info></list></nptrophy>`
	if code := post(body); code != 200 {
		t.Fatalf("native sync: %d", code)
	}
	// A second instance sees the same existing schema, not process-local state.
	games, err := (&Store{DB: pool}).Games(ctx, account)
	if err != nil || len(games) != 1 || len(games[0].Trophies) != 1 || games[0].Trophies[0].ID != 0 || games[0].Trophies[0].Grade != "B" {
		t.Fatalf("persisted native sync: %+v, %v", games, err)
	}
	if code := post(strings.ReplaceAll(body, "12:00:00Z", "13:00:00Z")); code != 200 {
		t.Fatalf("repeat sync: %d", code)
	}
	again, err := store.Games(ctx, account)
	if err != nil || again[0].Trophies[0].EarnedAt != games[0].Trophies[0].EarnedAt {
		t.Fatal("repeat sync changed first unlock", err)
	}
	resp, err := http.Get(server.URL + "/v1/users/" + account + "/trophies")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatal("private storage API exposed on native listener")
	}
	fake.SetStatus(account, accountv1.AccountStatus_ACCOUNT_STATUS_BANNED)
	if code := post(body); code != 401 {
		t.Fatalf("revoked account ticket accepted: %d", code)
	}
}
