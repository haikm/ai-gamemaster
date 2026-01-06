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

func (b *BritishAI) ProcessTurn(ctx context.Context, state GameState) string {
	englandPrompt := fmt.Sprintf("You are British High Command in WWI August 1914. Current situation: %s. Your fleet: %s. German fleet: %s. You only have the ships listed available. Do not make up other ships or units. What are your orders? Be concise.",
		state.Summary,
		strings.Join(state.BritishShips, ", "),
		strings.Join(state.GermanShips, ", "))
	britishOrder, err := b.Client.Call(ctx, "You are the British Admiral in command of the Royal Navy in WWI", englandPrompt)
	if err != nil {
		panic(err)
	}

	return britishOrder
}
