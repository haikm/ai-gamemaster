package game

import (
	"ai-gamemaster/domain"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

type Client interface {
	Call(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

type GameEngine struct {
	Client    Client
	Referee   domain.Referee
	BritishAI domain.BritishAI
	GameState *domain.GameState
}

func NewGameEngine(c Client, state domain.GameState) *GameEngine {
	engine := GameEngine{
		Client: c,
		Referee: domain.Referee{
			Client: c,
		},
		BritishAI: domain.BritishAI{
			Client: c,
		},
		GameState: &state,
	}

	return &engine
}

func (ge *GameEngine) ProcessTurn(ctx context.Context) {
	// Show current state
	fmt.Printf("\n=== Turn %d: %s ===\n", ge.GameState.Turn, ge.GameState.Date)
	fmt.Printf("%s\n\n", ge.GameState.Summary)
	fmt.Printf("British Fleet: %s\n", strings.Join(ge.GameState.BritishShips, ", "))
	fmt.Printf("German Fleet: %s\n\n", strings.Join(ge.GameState.GermanShips, ", "))

	// Get player input
	fmt.Println("Your orders (Germany):")
	reader := bufio.NewReader(os.Stdin)
	germanOrder, _ := reader.ReadString('\n')
	germanOrder = strings.TrimSpace(germanOrder)

	// Get British AI orders
	fmt.Println("\nBritish High Command is deciding...")
	britishOrder := ge.BritishAI.ProcessTurn(ctx, *ge.GameState)

	// Get referee decision
	fmt.Println("Referee is resolving turn...")
	refResult := ge.Referee.ProcessRefereeTurn(ctx, *ge.GameState, germanOrder, britishOrder)
	if refResult.RefereeResult.Narrative == "" {
		log.Printf("Warning: Could not parse referee result as JSON: %v\nUsing raw result.")
		// Fallback: just use raw result
		ge.GameState.Turn++
		ge.GameState.LastEvent = refResult.LastEvent
	} else {
		// Update ge.GameState based on parsed result
		ge.GameState.Turn++
		ge.GameState.Date = refResult.RefereeResult.NewDate
		ge.GameState.Summary = refResult.RefereeResult.NewSummary
		ge.GameState.LastEvent = refResult.RefereeResult.Narrative

		// Remove lost ships from British fleet
		ge.GameState.BritishShips = removeShips(ge.GameState.BritishShips, refResult.RefereeResult.BritishShipsLost)

		// Remove lost ships from German fleet
		ge.GameState.GermanShips = removeShips(ge.GameState.GermanShips, refResult.RefereeResult.GermanShipsLost)

		// Display losses
		if len(refResult.RefereeResult.BritishShipsLost) > 0 {
			fmt.Printf("British losses: %s\n", strings.Join(refResult.RefereeResult.BritishShipsLost, ", "))
		}
		if len(refResult.RefereeResult.GermanShipsLost) > 0 {
			fmt.Printf("German losses: %s\n", strings.Join(refResult.RefereeResult.GermanShipsLost, ", "))
		}
	}

	// Save ge.GameState
	newData, _ := json.MarshalIndent(ge.GameState, "", "  ")
	os.WriteFile("dumbstorage/game_state.json", newData, 0644)

	fmt.Println("State saved. Run again for next turn.")
}

// removeShips removes specified ships from a fleet
func removeShips(fleet []string, toRemove []string) []string {
	result := make([]string, 0, len(fleet))
	removeMap := make(map[string]bool)
	for _, ship := range toRemove {
		removeMap[ship] = true
	}
	for _, ship := range fleet {
		if !removeMap[ship] {
			result = append(result, ship)
		}
	}
	return result
}
