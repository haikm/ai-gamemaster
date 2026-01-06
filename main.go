package main

import (
	"ai-gamemaster/domain"
	"ai-gamemaster/game"
	"ai-gamemaster/gemini"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	_ "embed"
)

//go:embed dumbstorage/initial_state.json
var initialState []byte

func main() {
	ctx := context.Background()
	// Read game state

	var state domain.GameState

	gameState := initialState

	_, err := os.Stat("dumbstorage/game_state.json")
	if !errors.Is(err, os.ErrNotExist) {
		currentState, err := os.ReadFile("dumbstorage/game_state.json")
		if err != nil {
			log.Fatalf("can not read file, %v", err)
		}

		gameState = currentState
	}

	err = json.Unmarshal(gameState, &state)
	if err != nil {
		log.Fatalf("error unmarshaling game state: %v", err)
	}
	// new Client
	c, err := gemini.NewClient(ctx, "gemini-2.5-flash")
	if err != nil {
		log.Fatalf("error initialize client: %v", err)
	}
	//initialize game engine
	ge := game.NewGameEngine(c, state)

	ge.ProcessTurn(ctx)
}
