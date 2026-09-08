package internal

import (
	"fmt"
	"strings"
	"time"
)

const (
	IntentWrongLanguage = "wrong_audio_language"
	IntentMissingSubs   = "missing_subtitles"
	IntentWrongMatch    = "wrong_match"
	IntentQuality       = "quality_issue"
	IntentUnknown       = "unknown"

	StatusOpen     = "open"
	StatusResolved = "resolved"
	StatusFailed   = "failed"

	ActionReplaceMedia = "replace_media"
	ActionGenerateSubs = "generate_subtitles"
	ActionRematch      = "rematch_metadata"
	ActionNotifyAdmin  = "notify_admin"
)

// Ticket is a household issue the AI can classify and resolve.
type Ticket struct {
	ID         string      `json:"id"`
	Reporter   string      `json:"reporter,omitempty"`
	Subject    string      `json:"subject"`
	Body       string      `json:"body"`
	MediaID    string      `json:"media_id,omitempty"`
	Title      string      `json:"title,omitempty"`
	Status     string      `json:"status"`
	Intent     string      `json:"intent,omitempty"`
	Language   string      `json:"language,omitempty"`
	Action     string      `json:"action,omitempty"`
	CreatedAt  string      `json:"created_at"`
	ResolvedAt string      `json:"resolved_at,omitempty"`
	Error      string      `json:"error,omitempty"`
	Resolution *Resolution `json:"resolution,omitempty"`
}

// Resolution is the machine action a peer module can execute.
type Resolution struct {
	Action      string `json:"action"`
	MediaID     string `json:"media_id,omitempty"`
	Title       string `json:"title,omitempty"`
	Language    string `json:"language,omitempty"`
	RequestKind string `json:"request_kind,omitempty"` // movie, tv
	Summary     string `json:"summary"`
}

func classifyTicket(subject, body string) (intent, language, action string) {
	blob := strings.ToLower(subject + "\n" + body)
	switch {
	case containsAny(blob, "wrong language", "audio is in", "dubbed in", "not in english", "want the english"):
		return IntentWrongLanguage, extractLanguage(blob), ActionReplaceMedia
	case containsAny(blob, "no subtitle", "missing subtitle", "without subtitles", "can't find subtitles"):
		return IntentMissingSubs, extractLanguage(blob), ActionGenerateSubs
	case containsAny(blob, "wrong movie", "wrong show", "not the right", "misidentified", "wrong match"):
		return IntentWrongMatch, "", ActionRematch
	case containsAny(blob, "pixelated", "bad quality", "wrong audio", "out of sync"):
		return IntentQuality, "", ActionNotifyAdmin
	default:
		return IntentUnknown, "", ActionNotifyAdmin
	}
}

func resolveTicket(t Ticket) (Ticket, error) {
	if t.Intent == "" {
		t.Intent, t.Language, t.Action = classifyTicket(t.Subject, t.Body)
	}
	if t.Action == "" {
		_, _, t.Action = classifyTicket(t.Subject, t.Body)
	}
	if t.Intent == IntentUnknown {
		t.Status = StatusFailed
		t.Error = "could not classify ticket"
		return t, fmt.Errorf("could not classify ticket")
	}
	lang := t.Language
	if lang == "" {
		lang = "en"
	}
	title := firstNonEmpty(t.Title, t.Subject)
	t.Resolution = &Resolution{
		Action: t.Action, MediaID: t.MediaID, Title: title, Language: lang,
		RequestKind: "movie",
		Summary:     summaryFor(t.Intent, title, lang),
	}
	t.Status = StatusResolved
	t.ResolvedAt = time.Now().UTC().Format(time.RFC3339)
	return t, nil
}

func summaryFor(intent, title, lang string) string {
	switch intent {
	case IntentWrongLanguage:
		return fmt.Sprintf("Find a %s-language replacement for %s", lang, title)
	case IntentMissingSubs:
		return fmt.Sprintf("Generate %s subtitles for %s", lang, title)
	case IntentWrongMatch:
		return fmt.Sprintf("Rematch metadata for %s", title)
	default:
		return fmt.Sprintf("Admin review for %s", title)
	}
}

func extractLanguage(blob string) string {
	for _, pair := range []struct{ word, code string }{
		{"english", "en"}, {"spanish", "es"}, {"french", "fr"}, {"german", "de"},
		{"japanese", "ja"}, {"korean", "ko"}, {"portuguese", "pt"}, {"italian", "it"},
	} {
		if strings.Contains(blob, "want "+pair.word) || strings.Contains(blob, pair.word+" version") || strings.Contains(blob, "in "+pair.word) {
			if strings.Contains(blob, "want "+pair.word) || strings.Contains(blob, pair.word+" version") {
				return pair.code
			}
		}
	}
	if strings.Contains(blob, "want the english") || strings.Contains(blob, "not in english") {
		return "en"
	}
	return "en"
}

func containsAny(blob string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(blob, n) {
			return true
		}
	}
	return false
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
