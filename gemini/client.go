package gemini

import (
	"context"

	"google.golang.org/genai"
)

type Client struct {
	client *genai.Client
	model  string
}

func NewClient(ctx context.Context, model string) (*Client, error) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &Client{
		client: client,
		model:  model,
	}, nil
}

func (g *Client) Call(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	opts := &genai.GenerateContentConfig{}
	if systemPrompt != "" {
		opts.SystemInstruction = genai.Text(systemPrompt)[0]
	}

	result, err := g.client.Models.GenerateContent(
		ctx,
		g.model,
		genai.Text(userPrompt),
		opts,
	)
	if err != nil {
		return "", err
	}

	return result.Text(), nil
}
