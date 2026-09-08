package internal

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestOpenTicketHTTPResolvesWrongLanguage(t *testing.T) {
	m := New(Config{GRPCAddr: "127.0.0.1:0", HTTPAddr: "127.0.0.1:0"})
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Stop(context.Background()) })
	waitHealth(t, m.HTTPListenAddr())
	resp, err := http.Post("http://"+m.HTTPListenAddr()+"/v1/tickets", "application/json",
		strings.NewReader(`{"subject":"Wrong language","body":"audio is in Spanish, I want the English version","title":"Amelie","media_id":"m1"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	var got Ticket
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusResolved || got.Resolution == nil || got.Resolution.Action != ActionReplaceMedia {
		t.Fatalf("%s", raw)
	}
}

func waitHealth(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr + "/healthz")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == 200 {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("health timeout")
}
