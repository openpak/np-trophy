package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"openpak/np-trophy/internal/fakeachievements"
)

// One end-to-end test over HTTP and a fake achievements server: auth, validation, merge semantics
// (the fake mirrors the core's first-recorded-wins rule), ordering and account isolation. The real
// rule is tested in the achievements repo.
func TestTrophyAPI(t *testing.T) {
	ctx := context.Background()
	const achKey = "ach-key-0123456789abcdef0123456789abcdef"
	fake := fakeachievements.New(achKey)
	store, conn, err := DialAchievements("passthrough:///ach", achKey, fake.Serve(t)...)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = ctx
	key := "k-0123456789abcdef0123456789abcdef"
	srv := httptest.NewServer(Handler(store, key, slog.New(slog.NewTextHandler(io.Discard, nil))))
	defer srv.Close()
	const alice, bob = "0b7c3f4e-1d2a-4c5b-9e8f-112233445566", "aa7c3f4e-1d2a-4c5b-9e8f-112233445566"
	call := func(method, account, auth string, body any) (int, []Game) {
		var r io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			r = bytes.NewReader(b)
		}
		req, _ := http.NewRequest(method, srv.URL+"/v1/users/"+account+"/trophies", r)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out struct{ Games []Game }
		json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out.Games
	}
	bearer := "Bearer " + key
	set := func(comm string, unlocks ...Unlock) map[string]any {
		return map[string]any{"communication_id": comm, "trophies": unlocks}
	}
	for _, c := range []struct {
		name, method, account, auth string
		body                        any
		want                        int
	}{
		{"no key", "GET", alice, "", nil, 401},
		{"wrong key", "GET", alice, "Bearer nope", nil, 401},
		{"upper-case UUID", "GET", "0B7C3F4E-1D2A-4C5B-9E8F-112233445566", bearer, nil, 400},
		{"bad set", "POST", alice, bearer, set("npwr01234_00"), 400},
		{"bad id", "POST", alice, bearer, set("NPWR01234_00", Unlock{ID: 1024}), 400},
		{"bad grade", "POST", alice, bearer, set("NPWR01234_00", Unlock{ID: 1, Grade: "X"}), 400},
		{"method", "DELETE", alice, bearer, nil, 405},
	} {
		if got, _ := call(c.method, c.account, c.auth, c.body); got != c.want {
			t.Errorf("%s: status %d, want %d", c.name, got, c.want)
		}
	}
	if code, games := call("GET", alice, bearer, nil); code != 200 || games == nil || len(games) != 0 {
		t.Fatalf("empty account: %d %v", code, games)
	}
	call("POST", alice, bearer, set("NPWR02222_00", Unlock{ID: 3, EarnedAt: 300}, Unlock{ID: 0, EarnedAt: 100}))
	call("POST", alice, bearer, set("NPWR01111_00", Unlock{ID: 5, EarnedAt: 50, Grade: "P"}))
	// A later sync keeps the first unlock time and fills only a missing grade.
	code, games := call("POST", alice, bearer, set("NPWR02222_00", Unlock{ID: 0, EarnedAt: 999, Grade: "G"}, Unlock{ID: 7, EarnedAt: 700, Grade: "B"}))
	want := []Game{
		{CommunicationID: "NPWR01111_00", Trophies: []Unlock{{ID: 5, EarnedAt: 50, Grade: "P"}}},
		{CommunicationID: "NPWR02222_00", Trophies: []Unlock{{ID: 0, EarnedAt: 100, Grade: "G"}, {ID: 3, EarnedAt: 300}, {ID: 7, EarnedAt: 700, Grade: "B"}}},
	}
	if code != 200 || !equal(games, want) {
		t.Fatalf("merged: %d %+v", code, games)
	}
	call("POST", alice, bearer, set("NPWR02222_00", Unlock{ID: 0, EarnedAt: 1, Grade: "B"}))
	if _, games = call("GET", alice, bearer, nil); !equal(games, want) {
		t.Fatalf("first unlock and grade must stick: %+v", games)
	}
	// A wrong service key fails closed: the private API answers 500, never empty.
	bad, _, _ := DialAchievements("passthrough:///ach", "wrong-key-0123456789abcdef0123456789", fake.Serve(t)...)
	badSrv := httptest.NewServer(Handler(bad, key, slog.New(slog.NewTextHandler(io.Discard, nil))))
	defer badSrv.Close()
	if req, _ := http.NewRequest("GET", badSrv.URL+"/v1/users/"+alice+"/trophies", nil); true {
		req.Header.Set("Authorization", bearer)
		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != 500 {
			t.Fatalf("unreachable store: %v %v", err, resp)
		}
		resp.Body.Close()
	}
	if _, games = call("GET", bob, bearer, nil); len(games) != 0 {
		t.Fatalf("accounts must not share unlocks: %+v", games)
	}
}

func equal(a, b []Game) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y)
}
