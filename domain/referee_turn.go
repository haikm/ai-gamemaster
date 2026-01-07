package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type Ship struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

type Unit struct {
	Name      string `json:"name"`
	Commander string `json:"commander"`
	Ships     []Ship `json:"ships"`
	Location  string `json:"location"`
}

type GameState struct {
	Turn                       int      `json:"turn"`
	Date                       string   `json:"date"`
	Summary                    string   `json:"summary"`
	GermanSituation            string   `json:"german_situation"`
	BritishSituation           string   `json:"british_situation"`
	GermanIntelligenceReport   string   `json:"german_intelligence_report"`
	BritishIntelligenceReport  string   `json:"british_intelligence_report"`
	BritishUnits               []Unit   `json:"british_units"`
	GermanUnits                []Unit   `json:"german_units"`
	LastEvent                  string   `json:"last_event"`
}

type ShipStatusUpdate struct {
	Name      string `json:"name"`
	NewStatus string `json:"new_status"`
}

type UnitLocationUpdate struct {
	UnitName    string `json:"unit_name"`
	NewLocation string `json:"new_location"`
}

type RefereeResult struct {
	Narrative                    string               `json:"narrative"`
	BritishShipStatusUpdates     []ShipStatusUpdate   `json:"british_ship_status_updates"`
	GermanShipStatusUpdates      []ShipStatusUpdate   `json:"german_ship_status_updates"`
	BritishUnitLocationUpdates   []UnitLocationUpdate `json:"british_unit_location_updates"`
	GermanUnitLocationUpdates    []UnitLocationUpdate `json:"german_unit_location_updates"`
	NewSummary                   string               `json:"new_summary"`
	NewDate                      string               `json:"new_date"`
	NewGermanSituation           string               `json:"new_german_situation"`
	NewBritishSituation          string               `json:"new_british_situation"`
	NewGermanIntelligenceReport  string               `json:"new_german_intelligence_report"`
	NewBritishIntelligenceReport string               `json:"new_british_intelligence_report"`
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

func formatUnits(units []Unit) string {
	var parts []string
	for _, unit := range units {
		var shipNames []string
		for _, ship := range unit.Ships {
			// Only include non-sunk ships
			if ship.Status != "sunk" {
				shipNames = append(shipNames, fmt.Sprintf("%s (%s, %s)", ship.Name, ship.Type, ship.Status))
			}
		}
		// Only include units with at least one active ship
		if len(shipNames) > 0 {
			parts = append(parts, fmt.Sprintf("%s [%s] at %s: %s",
				unit.Name, unit.Commander, unit.Location, strings.Join(shipNames, ", ")))
		}
	}
	return strings.Join(parts, "\n")
}

func (r *Referee) ProcessRefereeTurn(ctx context.Context, state GameState, germanOrder, britishOrder string) Result {
	refPrompt := fmt.Sprintf(`Current game state:
Turn: %d
Date: %s
Summary: %s

German Situation: %s
British Situation: %s

German Intelligence: %s
British Intelligence: %s

German Forces:
%s

British Forces:
%s

Germany orders: %s
Britain orders: %s

Resolve this turn and respond ONLY with valid JSON in this exact format:
{
  "narrative": "Brief objective description of what actually happened",
  "british_ship_status_updates": [{"name": "ship name", "new_status": "operational/damaged/heavily damaged/sunk"}],
  "german_ship_status_updates": [{"name": "ship name", "new_status": "operational/damaged/heavily damaged/sunk"}],
  "british_unit_location_updates": [{"unit_name": "unit name", "new_location": "new location"}],
  "german_unit_location_updates": [{"unit_name": "unit name", "new_location": "new location"}],
  "new_summary": "Updated overall situation for next turn",
  "new_date": "YYYY-MM-DD format, a few days after %s",
  "new_german_situation": "Updated German force status and readiness",
  "new_british_situation": "Updated British force status and readiness",
  "new_german_intelligence_report": "What German intelligence reports about British forces (fog of war - uncertain, incomplete)",
  "new_british_intelligence_report": "What British intelligence reports about German forces (fog of war - uncertain, incomplete)"
}

Be concise and realistic. Only reference ships and units from those listed above.
Empty arrays if no changes.
Ship status can be: operational, damaged, heavily damaged, sunk.
IMPORTANT: Update unit locations whenever units move. Use specific naval location names (ports, sea areas, coordinates).
For intelligence reports: Each side should only know what they could realistically observe - uncertain enemy positions, spotted ships, reconnaissance reports with limited accuracy. They know their own forces clearly.`,
		state.Turn, state.Date, state.Summary,
		state.GermanSituation, state.BritishSituation,
		state.GermanIntelligenceReport, state.BritishIntelligenceReport,
		formatUnits(state.GermanUnits),
		formatUnits(state.BritishUnits),
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
