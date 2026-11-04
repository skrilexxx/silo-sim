// Package narrate turns a run's chronicle into a story written by Claude
// (docs/narrator.md, T17.2). It is the only package that imports the
// Anthropic SDK (D14) or talks to the network; it sends the chronicle and
// nothing else, and writes one file, story.md, into the run folder.
package narrate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
	"github.com/anthropics/anthropic-sdk-go/v2/shared/constant"

	"github.com/skrilexxx/silo-sim/internal/chronicle"
)

// Model is the default model (docs/narrator.md, The story).
const Model = "claude-opus-5-5"

// maxTokens leaves room for the thinking and a story of up to about 3,000
// words; the request is streamed, so a high cap costs nothing unused.
const maxTokens = 64000

// System is the fixed system prompt, quoted in docs/narrator.md.
const System = `You are the archivist of Silo 18, an underground silo of 144 floors under the Pact, writing the record of its years from the chronicle you are given. Write it as an account a reader in the silo could keep: in eras, under headings, in plain prose.

Use only what the chronicle says. Every event, year and number in your account must be in it; when you round a number, say "about". Do not add events, causes, motives or people it does not give. Where it is silent, say so or say nothing. People have no names: call them by their role and floor or zone ("a cleaner from the Down Deep", "the new Head of IT"). Do not use characters, places or events from the television series Silo or the Wool books.

Aim for 1,000 to 2,500 words, in Markdown, with a heading per era and a short closing paragraph on how the silo stands at the end.`

// Write narrates the run folder dir with model and writes dir/story.md,
// headed by the model and the day (now). On any error it writes nothing.
// opts reach the API client (tests point it at a fake server).
func Write(ctx context.Context, dir, model string, now time.Time, opts ...option.RequestOption) error {
	var chron bytes.Buffer
	if err := chronicle.Write(&chron, dir); err != nil {
		return err
	}
	text, err := story(ctx, anthropic.NewClient(opts...), model, chron.String())
	if err != nil {
		return err
	}
	head := fmt.Sprintf("*Written by %s on %s from this run's chronicle. The model may err; the chronicle below is the record.*\n\n", model, now.Format("2006-01-02"))
	return os.WriteFile(filepath.Join(dir, "story.md"), []byte(head+strings.TrimSpace(text)+"\n"), 0o644)
}

// story sends one streamed request: the system prompt and the chronicle,
// with the server-side refusal fallback on (Haiku has none), and returns
// the reply's text.
func story(ctx context.Context, client anthropic.Client, model, chron string) (string, error) {
	params := anthropic.BetaMessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: maxTokens,
		System:    []anthropic.BetaTextBlockParam{{Text: System}},
		Messages:  []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(chron))},
	}
	if !strings.HasPrefix(model, "claude-haiku") {
		params.Betas = []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01}
		params.Fallbacks = anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()}
	}
	stream := client.Beta.Messages.NewStreaming(ctx, params)
	msg := anthropic.BetaMessage{}
	for stream.Next() {
		if err := msg.Accumulate(stream.Current()); err != nil {
			return "", err
		}
	}
	if err := stream.Err(); err != nil {
		var apiErr *anthropic.Error
		if errors.As(err, &apiErr) && apiErr.StatusCode == 401 {
			return "", fmt.Errorf("%w\nset ANTHROPIC_API_KEY, or run `ant auth login`", err)
		}
		return "", err
	}
	switch msg.StopReason {
	case anthropic.BetaStopReasonRefusal:
		return "", fmt.Errorf("Claude declined to write the story (%s)", msg.StopDetails.Category)
	case anthropic.BetaStopReasonMaxTokens:
		return "", errors.New("the story was cut off at the token limit")
	}
	var b strings.Builder
	for _, block := range msg.Content {
		if t, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			b.WriteString(t.Text)
		}
	}
	if strings.TrimSpace(b.String()) == "" {
		return "", errors.New("the reply held no text")
	}
	return b.String(), nil
}
