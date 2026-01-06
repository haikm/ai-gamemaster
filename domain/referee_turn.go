package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type GameState struct {
	Turn         int      `json:"turn"`
	Date         string   `json:"date"`
	Summary      string   `json:"summary"`
	BritishShips []string `json:"british_ships"`
	GermanShips  []string `json:"german_ships"`
	LastEvent    string   `json:"last_event"`
}

type RefereeResult struct {
	Narrative        string   `json:"narrative"`
	BritishShipsLost []string `json:"british_ships_lost"`
	GermanShipsLost  []string `json:"german_ships_lost"`
	NewSummary       string   `json:"new_summary"`
	NewDate          string   `json:"new_date"`
}

type Result struct {
	RefereeResult RefereeResult
	LastEvent     string
}

type Referee struct {
	Client interface {
		Call(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	}
}

func (r *Referee) ProcessRefereeTurn(ctx context.Context, state GameState, germanOrder, britishOrder string) Result {
	refPrompt := fmt.Sprintf(`Current game state:
Turn: %d
Date: %s
Summary: %s
British Fleet: %s
German Fleet: %s

Germany orders: %s
Britain orders: %s

Resolve this turn and respond ONLY with valid JSON in this exact format:
{
  "narrative": "Brief description of what happened",
  "british_ships_lost": ["ship name if any lost"],
  "german_ships_lost": ["ship name if any lost"],
  "new_summary": "Updated situation for next turn",
  "new_date": "YYYY-MM-DD format, a few days after %s"
}

Be concise and realistic. Only reference ships from the fleets listed above. Empty arrays if no ships lost.`,
		state.Turn, state.Date, state.Summary,
		strings.Join(state.BritishShips, ", "),
		strings.Join(state.GermanShips, ", "),
		germanOrder, britishOrder, state.Date)

	result, err := r.Client.Call(ctx, "You are a realistic WWI naval wargame referee. Evaluate orders and determine outcomes based on historical naval doctrine and capabilities.", refPrompt)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\n=== TURN RESULT ===\n%s\n\n", result)

	// Parse referee result
	var refResult RefereeResult
	cleanJSON := extractJSON(result)
	err = json.Unmarshal([]byte(cleanJSON), &refResult)
	if err != nil {
		return Result{
			RefereeResult: RefereeResult{},
			LastEvent:     result,
		}
	}

	return Result{
		RefereeResult: refResult,
		LastEvent:     result,
	}
}
