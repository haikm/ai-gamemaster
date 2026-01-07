package domain

import (
	"context"
	"fmt"
	"strings"
)

type BritishAI struct {
	Client interface {
		Call(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	}
}

func formatUnitsForAI(units []Unit) string {
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

func (b *BritishAI) ProcessTurn(ctx context.Context, state GameState) string {
	englandPrompt := fmt.Sprintf(`You are British High Command in WWI.

Current situation: %s

Your situation: %s

Your forces:
%s

Intelligence report on German forces:
%s

What are your orders? Be concise and realistic. Only reference the units and ships you have available.`,
		state.Summary,
		state.BritishSituation,
		formatUnitsForAI(state.BritishUnits),
		state.BritishIntelligenceReport)

	britishOrder, err := b.Client.Call(ctx, "You are the British Admiral in command of the Royal Navy in WWI", englandPrompt)
	if err != nil {
		panic(err)
	}

	return britishOrder
}
