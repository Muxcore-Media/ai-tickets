package internal

import "testing"

func TestClassifyWrongLanguage(t *testing.T) {
	intent, lang, action := classifyTicket("Language wrong", "The audio is in Spanish, I want the English version")
	if intent != IntentWrongLanguage || lang != "en" || action != ActionReplaceMedia {
		t.Fatalf("%s %s %s", intent, lang, action)
	}
}

func TestResolveProducesReplacement(t *testing.T) {
	got, err := resolveTicket(Ticket{
		ID: "t1", Subject: "Wrong language", Body: "audio is in French, want the English version",
		MediaID: "m1", Title: "Amelie",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusResolved || got.Resolution == nil || got.Resolution.Action != ActionReplaceMedia {
		t.Fatalf("%+v", got)
	}
	if got.Resolution.Language != "en" || got.Resolution.Title != "Amelie" {
		t.Fatalf("resolution %+v", got.Resolution)
	}
}

func TestUnknownTicketFailsClosed(t *testing.T) {
	_, err := resolveTicket(Ticket{Subject: "hello", Body: "just saying hi"})
	if err == nil {
		t.Fatal("expected classify failure")
	}
}
