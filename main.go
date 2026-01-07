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
	"os/signal"
	"syscall"

	_ "embed"
)

//go:embed dumbstorage/initial_state.json
var initialState []byte

func main() {
	// Set up context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handler for SIGINT (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Handle signals in a goroutine
	go func() {
		<-sigChan
		fmt.Println("\n\nReceived interrupt signal. Exiting...")
		cancel()
		os.Exit(0)
	}()

	// Initialize client once
	c, err := gemini.NewClient(ctx, "gemini-3-flash-preview")
	if err != nil {
		log.Fatalf("error initialize client: %v", err)
	}

	// Game loop
	for {
		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Load game state
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

		// Initialize game engine with current state
		ge := game.NewGameEngine(c, state)

		// Process turn - returns false if user types "exit"
		if !ge.ProcessTurn(ctx) {
			return
		}
	}
}
