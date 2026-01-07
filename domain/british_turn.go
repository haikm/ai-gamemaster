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

Your forces (THESE ARE THE ONLY SHIPS YOU HAVE):
%s

Intelligence report on German forces:
%s

Give your orders for the turn. Be concise and realistic.

CRITICAL RULES:
- You can ONLY use ships and units explicitly listed above under "Your forces"
- DO NOT mention, reference, or give orders to ANY other ships, destroyers, cruisers, or units
- If you mention ANY ship not in your force list above, your orders will be invalid
- There are NO reinforcements, NO other squadrons, NO additional ships available`,
		state.Summary,
		state.BritishSituation,
		formatUnitsForAI(state.BritishUnits),
		state.BritishIntelligenceReport)

	britishOrder, err := b.Client.Call(ctx, "You are the British Admiral in command of the Royal Navy in WWI. You command ONLY the ships explicitly listed in your prompt.", englandPrompt)
	if err != nil {
		panic(err)
	}

	return britishOrder
}
