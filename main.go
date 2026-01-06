package main

import (
	"ai-gamemaster/gemini"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	_ "embed"
)

//go:embed dumbstorage/initial_state.json
var initialState []byte

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

func main() {
	ctx := context.Background()
	// Read game state

	var state GameState

	gameState := initialState

	_, err := os.Stat("dumbstorage/game_state.json")
	if !errors.Is(err, os.ErrNotExist) {
		currentState, err := os.ReadFile("dumbstorage/game_state.json")
		if err != nil {
			log.Fatalf("can not read file, %v", err)
		}

		gameState = currentState
	}

	json.Unmarshal(gameState, &state)

	// Show current state
	fmt.Printf("\n=== Turn %d: %s ===\n", state.Turn, state.Date)
	fmt.Printf("%s\n\n", state.Summary)
	fmt.Printf("British Fleet: %s\n", strings.Join(state.BritishShips, ", "))
	fmt.Printf("German Fleet: %s\n\n", strings.Join(state.GermanShips, ", "))

	// Initialize AI client
	client, err := gemini.NewClient(ctx, "gemini-2.5-flash")
	if err != nil {
		log.Fatalf("error creating gemini client, %v", err)
	}

	// Get player input
	fmt.Println("Your orders (Germany):")
	reader := bufio.NewReader(os.Stdin)
	germanOrder, _ := reader.ReadString('\n')
	germanOrder = strings.TrimSpace(germanOrder)

	// Get British AI orders
	fmt.Println("\nBritish High Command is deciding...")
	englandPrompt := fmt.Sprintf("You are British High Command in WWI August 1914. Current situation: %s. Your fleet: %s. German fleet: %s. You only have the ships listed available. Do not make up other ships or units. What are your orders? Be concise.",
		state.Summary,
		strings.Join(state.BritishShips, ", "),
		strings.Join(state.GermanShips, ", "))
	britishOrder, err := client.Call(ctx, "You are the British Admiral in command of the Royal Navy in WWI", englandPrompt)
	if err != nil {
		panic(err)
	}
	fmt.Printf("British orders: %s\n\n", britishOrder)

	// Get referee decision
	fmt.Println("Referee is resolving turn...")
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

	result, err := client.Call(ctx, "You are a realistic WWI naval wargame referee. Evaluate orders and determine outcomes based on historical naval doctrine and capabilities.", refPrompt)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\n=== TURN RESULT ===\n%s\n\n", result)

	// Parse referee result
	var refResult RefereeResult
	cleanJSON := extractJSON(result)
	if err := json.Unmarshal([]byte(cleanJSON), &refResult); err != nil {
		log.Printf("Warning: Could not parse referee result as JSON: %v\nUsing raw result.", err)
		// Fallback: just use raw result
		state.Turn++
		state.LastEvent = result
	} else {
		// Update state based on parsed result
		state.Turn++
		state.Date = refResult.NewDate
		state.Summary = refResult.NewSummary
		state.LastEvent = refResult.Narrative

		// Remove lost ships from British fleet
		state.BritishShips = removeShips(state.BritishShips, refResult.BritishShipsLost)

		// Remove lost ships from German fleet
		state.GermanShips = removeShips(state.GermanShips, refResult.GermanShipsLost)

		// Display losses
		if len(refResult.BritishShipsLost) > 0 {
			fmt.Printf("British losses: %s\n", strings.Join(refResult.BritishShipsLost, ", "))
		}
		if len(refResult.GermanShipsLost) > 0 {
			fmt.Printf("German losses: %s\n", strings.Join(refResult.GermanShipsLost, ", "))
		}
	}

	// Save state
	newData, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile("dumbstorage/game_state.json", newData, 0644)

	fmt.Println("State saved. Run again for next turn.")
}
