package domain

import "strings"

// extractJSON strips markdown code fences from JSON response
func extractJSON(text string) string {
	text = strings.TrimSpace(text)

	// Check if it starts with ```json
	if strings.HasPrefix(text, "```json") {
		// Remove opening ```json
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimSpace(text)

		// Remove closing ```
		if idx := strings.LastIndex(text, "```"); idx != -1 {
			text = text[:idx]
		}
		text = strings.TrimSpace(text)
	}

	return text
}
