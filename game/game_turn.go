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

func displayUnits(units []domain.Unit) {
	for _, unit := range units {
		// Only show units with at least one non-sunk ship
		activeShips := []domain.Ship{}
		for _, ship := range unit.Ships {
			if ship.Status != "sunk" {
				activeShips = append(activeShips, ship)
			}
		}
		if len(activeShips) == 0 {
			continue
		}

		fmt.Printf("%s [%s] at %s\n", unit.Name, unit.Commander, unit.Location)
		for _, ship := range activeShips {
			fmt.Printf("  - %s (%s) [%s]\n", ship.Name, ship.Type, ship.Status)
		}
	}
}

// countActiveShips returns the number of non-sunk ships in units
func countActiveShips(units []domain.Unit) int {
	count := 0
	for _, unit := range units {
		for _, ship := range unit.Ships {
			if ship.Status != "sunk" {
				count++
			}
		}
	}
	return count
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

func (ge *GameEngine) ProcessTurn(ctx context.Context) bool {
	// Show current state
	fmt.Printf("\n=== Turn %d: %s ===\n", ge.GameState.Turn, ge.GameState.Date)
	fmt.Printf("%s\n\n", ge.GameState.Summary)

	// Display German situation (what the German player knows)
	fmt.Printf("=== Your Situation ===\n%s\n\n", ge.GameState.GermanSituation)

	// Display German forces
	fmt.Println("=== Your Forces ===")
	displayUnits(ge.GameState.GermanUnits)

	// Display German intelligence about British forces
	fmt.Printf("\n=== Intelligence Report ===\n%s\n\n", ge.GameState.GermanIntelligenceReport)

	// Get player input
	fmt.Println("Your orders (Germany):")
	reader := bufio.NewReader(os.Stdin)
	germanOrder, _ := reader.ReadString('\n')
	germanOrder = strings.TrimSpace(germanOrder)

	// Check for exit command
	if strings.ToLower(germanOrder) == "exit" {
		fmt.Println("Exiting game...")
		return false
	}

	// Get British AI orders
	fmt.Println("\nBritish High Command is deciding...")
	britishOrder := ge.BritishAI.ProcessTurn(ctx, *ge.GameState)

	// Get referee decision
	fmt.Println("Referee is resolving turn...")
	refResult := ge.Referee.ProcessRefereeTurn(ctx, *ge.GameState, germanOrder, britishOrder)
	if refResult.RefereeResult.Narrative == "" {
		log.Printf("Warning: Could not parse referee result as JSON. Using raw result.")
		// Fallback: just use raw result
		ge.GameState.Turn++
		ge.GameState.LastEvent = refResult.LastEvent
	} else {
		// Update ge.GameState based on parsed result
		ge.GameState.Turn++
		ge.GameState.Date = refResult.RefereeResult.NewDate
		ge.GameState.Summary = refResult.RefereeResult.NewSummary
		ge.GameState.LastEvent = refResult.RefereeResult.Narrative
		ge.GameState.GermanSituation = refResult.RefereeResult.NewGermanSituation
		ge.GameState.BritishSituation = refResult.RefereeResult.NewBritishSituation
		ge.GameState.GermanIntelligenceReport = refResult.RefereeResult.NewGermanIntelligenceReport
		ge.GameState.BritishIntelligenceReport = refResult.RefereeResult.NewBritishIntelligenceReport

		// Update ship statuses for British fleet
		ge.GameState.BritishUnits = updateShipStatuses(ge.GameState.BritishUnits, refResult.RefereeResult.BritishShipStatusUpdates)

		// Update ship statuses for German fleet
		ge.GameState.GermanUnits = updateShipStatuses(ge.GameState.GermanUnits, refResult.RefereeResult.GermanShipStatusUpdates)

		// Update unit locations for British fleet
		ge.GameState.BritishUnits = updateUnitLocations(ge.GameState.BritishUnits, refResult.RefereeResult.BritishUnitLocationUpdates)

		// Update unit locations for German fleet
		ge.GameState.GermanUnits = updateUnitLocations(ge.GameState.GermanUnits, refResult.RefereeResult.GermanUnitLocationUpdates)

		// Display what the German commander knows (fog of war)
		fmt.Printf("\n=== Turn %d Results ===\n", ge.GameState.Turn-1)
		fmt.Printf("\n=== Your Intelligence Report ===\n%s\n", refResult.RefereeResult.NewGermanIntelligenceReport)
		fmt.Printf("\n=== Your Situation ===\n%s\n", refResult.RefereeResult.NewGermanSituation)

		// Check victory conditions
		germanActive := countActiveShips(ge.GameState.GermanUnits)
		britishActive := countActiveShips(ge.GameState.BritishUnits)

		if germanActive == 0 && britishActive == 0 {
			fmt.Println("\n\n=== MUTUAL DESTRUCTION ===")
			fmt.Println("Both fleets have been annihilated. The war at sea ends in a pyrrhic stalemate.")
			return false
		} else if germanActive == 0 {
			fmt.Println("\n\n=== BRITISH VICTORY ===")
			fmt.Println("The German fleet has been destroyed. Britannia rules the waves!")
			return false
		} else if britishActive == 0 {
			fmt.Println("\n\n=== GERMAN VICTORY ===")
			fmt.Println("The British fleet has been destroyed. The Kaiserliche Marine is triumphant!")
			return false
		}
	}

	// Save ge.GameState
	newData, _ := json.MarshalIndent(ge.GameState, "", "  ")
	os.WriteFile("dumbstorage/game_state.json", newData, 0644)

	fmt.Println("\nState saved.")
	return true
}

// updateUnitLocations updates the locations of units
func updateUnitLocations(units []domain.Unit, updates []domain.UnitLocationUpdate) []domain.Unit {
	updateMap := make(map[string]string)
	for _, update := range updates {
		updateMap[update.UnitName] = update.NewLocation
	}

	result := make([]domain.Unit, 0, len(units))
	for _, unit := range units {
		if newLocation, ok := updateMap[unit.Name]; ok {
			unit.Location = newLocation
		}
		result = append(result, unit)
	}
	return result
}

// updateShipStatuses updates the status of ships in units
func updateShipStatuses(units []domain.Unit, updates []domain.ShipStatusUpdate) []domain.Unit {
	updateMap := make(map[string]string)
	for _, update := range updates {
		updateMap[update.Name] = update.NewStatus
	}

	result := make([]domain.Unit, 0, len(units))
	for _, unit := range units {
		newShips := make([]domain.Ship, len(unit.Ships))
		for i, ship := range unit.Ships {
			newShips[i] = ship
			if newStatus, ok := updateMap[ship.Name]; ok {
				newShips[i].Status = newStatus
			}
		}
		unit.Ships = newShips
		result = append(result, unit)
	}
	return result
}
