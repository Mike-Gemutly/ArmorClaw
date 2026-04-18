package ai

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

const (
	// DefaultCompactionThreshold is the token count at which compaction triggers.
	DefaultCompactionThreshold = 100000

	// compactionPrompt is the system instruction for the summarization LLM call.
	compactionPrompt = `You are a session transcript compactor. Compress the conversation below into a concise summary preserving:
- Key facts and data points mentioned
- Decisions made and their rationales
- Current task state and any open questions
- Constraints or requirements identified

Do NOT include pleasantries, acknowledgments, or filler. Output only the factual summary.`

	// fallbackSummary is used when the LLM call fails.
	fallbackSummary = "Prior session history was compacted but the summary could not be generated."
)

// EstimateMessageTokens returns an approximate token count for a slice of messages.
// Uses the same ~4 chars/token heuristic as the rest of the bridge (CharsPerToken).
func EstimateMessageTokens(messages []Message) int {
	total := 0
	for _, msg := range messages {
		// Count role + separator + content
		total += (len(msg.Role) + 2 + len(msg.Content)) / CharsPerToken
	}
	return total
}

// ShouldCompact returns true when the estimated token count exceeds the threshold.
func ShouldCompact(messages []Message, threshold int) bool {
	if threshold <= 0 {
		threshold = DefaultCompactionThreshold
	}
	return EstimateMessageTokens(messages) >= threshold
}

// CompactHistory compacts a message slice by requesting an LLM summary when the
// estimated token count exceeds the threshold. It returns either the original
// messages (when below threshold) or a compacted slice containing the summary as
// a single system message.
//
// The client parameter is used for the summarization call; it may be nil, in which
// case CompactHistory falls back to a local truncation strategy.
func CompactHistory(ctx context.Context, client AIClient, messages []Message, threshold int) ([]Message, error) {
	if threshold <= 0 {
		threshold = DefaultCompactionThreshold
	}

	estimated := EstimateMessageTokens(messages)
	if estimated < threshold {
		return messages, nil
	}

	slog.Info("compacting session transcript",
		"estimated_tokens", estimated,
		"threshold", threshold,
		"message_count", len(messages),
	)

	summary, err := summarizeMessages(ctx, client, messages)
	if err != nil {
		slog.Warn("LLM summarization failed, using truncation fallback", "error", err)
		summary = truncationFallback(messages)
	}

	compacted := []Message{
		{
			Role:    "system",
			Content: fmt.Sprintf("[Compacted session summary]\n%s", summary),
		},
	}

	slog.Info("session transcript compacted",
		"original_tokens", estimated,
		"compacted_tokens", EstimateMessageTokens(compacted),
		"original_messages", len(messages),
		"compacted_messages", len(compacted),
	)

	return compacted, nil
}

// summarizeMessages calls the LLM to produce a summary of the given messages.
func summarizeMessages(ctx context.Context, client AIClient, messages []Message) (string, error) {
	if client == nil {
		return "", fmt.Errorf("no AI client available for summarization")
	}

	// Build the summarization request: system prompt + concatenated transcript.
	var b strings.Builder
	for _, msg := range messages {
		b.WriteString(msg.Role)
		b.WriteString(": ")
		b.WriteString(msg.Content)
		b.WriteString("\n")
	}

	req := ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []Message{
			{Role: "system", Content: compactionPrompt},
			{Role: "user", Content: b.String()},
		},
		Temperature: 0.0,
		MaxTokens:   1024,
	}

	resp, err := client.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("summarization chat call failed: %w", err)
	}

	if resp.Content == "" {
		return fallbackSummary, nil
	}

	return resp.Content, nil
}

// truncationFallback keeps only the first and last messages when LLM is unavailable.
func truncationFallback(messages []Message) string {
	if len(messages) == 0 {
		return fallbackSummary
	}

	var b strings.Builder
	b.WriteString("Session had ")
	b.WriteString(fmt.Sprintf("%d", len(messages)))
	b.WriteString(" messages. Key excerpts:\n\n")

	// Keep first message for context.
	if len(messages) > 0 {
		b.WriteString("[First] ")
		truncateContent(&b, messages[0], 500)
		b.WriteString("\n")
	}

	// Keep last message for current state.
	if len(messages) > 1 {
		b.WriteString("[Last] ")
		truncateContent(&b, messages[len(messages)-1], 500)
		b.WriteString("\n")
	}

	return b.String()
}

func truncateContent(b *strings.Builder, msg Message, maxLen int) {
	if len(msg.Content) <= maxLen {
		b.WriteString(msg.Role)
		b.WriteString(": ")
		b.WriteString(msg.Content)
		return
	}
	b.WriteString(msg.Role)
	b.WriteString(": ")
	b.WriteString(msg.Content[:maxLen])
	b.WriteString("...")
}
