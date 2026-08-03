package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"open-pptx/internal/engine"
)

const systemPrompt = `You are an expert AI presentation designer and co-author for open-pptx.
Your job is to generate visually stunning, modern, clear presentation slides based on user prompts.

SLIDE CANVAS SPECIFICATIONS:
- Dimensions: 960px width x 540px height (16:9 widescreen format).
- Elements must fit comfortably inside (X: 0..960, Y: 0..540).
- Keep text elements wide enough (W: 400..800) to avoid awkward word wrapping.

SLIDE STRUCTURE:
- Slide title: Y=60..100, FontSize=40..52, FontWeight="bold", Color="#0F172A", TextAlign="left" or "center".
- Subtitle / Body text: Y=140..400, FontSize=18..24, Color="#334155".
- Shape elements: Used as card backgrounds or visual accent lines.
  - Card shape: BgColor="#F1F5F9", BorderRadius=12, W=240..300, H=200..300.
- Code elements: Type="code", FontSize=14..16, BgColor="#F1F5F9", Color="#0F172A", W=600..800, H=250..350.

COLOR PALETTE (Modern Light Canvas Theme):
- Background: #FFFFFF (clean white)
- Cards/Surfaces: #F1F5F9 or #F8FAFC
- Primary Accent: #2563EB (blue)
- Secondary Accent: #7C3AED (purple)
- Heading text: #0F172A (dark slate)
- Subtitle/Body text: #334155 or #64748B

OUTPUT FORMAT:
You MUST respond with a valid JSON object only. No markdown formatting, no code blocks (unless inside the JSON string), no commentary.

When generating a DECK, output JSON matching:
{
  "title": "Deck Title",
  "slides": [
    {
      "layout": "title" | "content" | "cards" | "code",
      "bgColor": "#FFFFFF",
      "elements": [
        {
          "id": "el-1",
          "type": "text" | "shape" | "code",
          "content": "Text or code content",
          "position": {"x": 80, "y": 80, "w": 800, "h": 80},
          "style": {
            "fontSize": 48,
            "fontWeight": "bold",
            "color": "#0F172A",
            "textAlign": "center",
            "bgColor": "#F1F5F9",
            "borderRadius": 12
          },
          "zIndex": 1
        }
      ]
    }
  ]
}

When editing a single SLIDE, output JSON matching:
{
  "bgColor": "#0F172A",
  "elements": [...]
}
`

// Agent provides slide generation and editing capabilities.
type Agent struct {
	client *Client
}

// NewAgent creates a new AI Agent.
func NewAgent(cfg Config) *Agent {
	return &Agent{
		client: NewClient(cfg),
	}
}

type GeneratedDeckJSON struct {
	Title  string         `json:"title"`
	Slides []engine.Slide `json:"slides"`
}

// GenerateDeck creates a new deck from a natural language topic.
func (a *Agent) GenerateDeck(ctx context.Context, prompt string) (*engine.Deck, error) {
	userPrompt := fmt.Sprintf("Create a 4-to-6 slide presentation deck about: '%s'. Return valid JSON.", prompt)

	jsonStr, err := a.client.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var gen GeneratedDeckJSON
	if err := json.Unmarshal([]byte(jsonStr), &gen); err != nil {
		return nil, fmt.Errorf("parse AI response: %w\nResponse was: %s", err, jsonStr)
	}

	deck := engine.NewDeck()
	if gen.Title != "" {
		deck.Meta.Title = gen.Title
	} else {
		deck.Meta.Title = prompt
	}

	if len(gen.Slides) > 0 {
		// Ensure IDs on all generated elements and slides
		for sIdx := range gen.Slides {
			gen.Slides[sIdx].ID = fmt.Sprintf("slide-ai-%d", sIdx+1)
			if gen.Slides[sIdx].BgColor == "" {
				gen.Slides[sIdx].BgColor = "#FFFFFF"
			}
			for eIdx := range gen.Slides[sIdx].Elements {
				gen.Slides[sIdx].Elements[eIdx].ID = fmt.Sprintf("el-ai-%d-%d", sIdx+1, eIdx+1)
			}
		}
		deck.Slides = gen.Slides
	}

	return &deck, nil
}

// GenerateDeckStream streams reasoning and content while building a presentation.
func (a *Agent) GenerateDeckStream(ctx context.Context, prompt string, callback func(chunk StreamChunk)) (*engine.Deck, error) {
	userPrompt := fmt.Sprintf("Create a 4-to-6 slide presentation deck about: '%s'. Return valid JSON.", prompt)

	jsonStr, err := a.client.CompleteStream(ctx, systemPrompt, userPrompt, callback)
	if err != nil {
		return nil, err
	}

	var gen GeneratedDeckJSON
	if err := json.Unmarshal([]byte(jsonStr), &gen); err != nil {
		return nil, fmt.Errorf("parse AI response: %w\nResponse was: %s", err, jsonStr)
	}

	deck := engine.NewDeck()
	if gen.Title != "" {
		deck.Meta.Title = gen.Title
	} else {
		deck.Meta.Title = prompt
	}

	if len(gen.Slides) > 0 {
		for sIdx := range gen.Slides {
			gen.Slides[sIdx].ID = fmt.Sprintf("slide-ai-%d", sIdx+1)
			if gen.Slides[sIdx].BgColor == "" {
				gen.Slides[sIdx].BgColor = "#FFFFFF"
			}
			for eIdx := range gen.Slides[sIdx].Elements {
				gen.Slides[sIdx].Elements[eIdx].ID = fmt.Sprintf("el-ai-%d-%d", sIdx+1, eIdx+1)
			}
		}
		deck.Slides = gen.Slides
	}

	return &deck, nil
}

// AddSlide generates and appends a single slide to the deck based on a prompt.
func (a *Agent) AddSlide(ctx context.Context, currentDeck *engine.Deck, prompt string) (*engine.Slide, error) {
	deckContext, _ := json.Marshal(currentDeck.Meta)
	userPrompt := fmt.Sprintf("Deck metadata: %s\n\nGenerate ONE new slide based on prompt: '%s'. Return valid JSON for the slide elements and background.", string(deckContext), prompt)

	jsonStr, err := a.client.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var slide engine.Slide
	if err := json.Unmarshal([]byte(jsonStr), &slide); err != nil {
		return nil, fmt.Errorf("parse slide response: %w", err)
	}

	slide.ID = fmt.Sprintf("slide-ai-%d", len(currentDeck.Slides)+1)
	if slide.BgColor == "" {
		slide.BgColor = "#FFFFFF"
	}
	for i := range slide.Elements {
		slide.Elements[i].ID = fmt.Sprintf("el-ai-%d", i+1)
	}

	return &slide, nil
}

// AddSlideStream streams reasoning and content while creating a single slide.
func (a *Agent) AddSlideStream(ctx context.Context, currentDeck *engine.Deck, prompt string, callback func(chunk StreamChunk)) (*engine.Slide, error) {
	deckContext, _ := json.Marshal(currentDeck.Meta)
	userPrompt := fmt.Sprintf("Deck metadata: %s\n\nGenerate ONE new slide based on prompt: '%s'. Return valid JSON for the slide elements and background.", string(deckContext), prompt)

	jsonStr, err := a.client.CompleteStream(ctx, systemPrompt, userPrompt, callback)
	if err != nil {
		return nil, err
	}

	var slide engine.Slide
	if err := json.Unmarshal([]byte(jsonStr), &slide); err != nil {
		return nil, fmt.Errorf("parse slide response: %w", err)
	}

	slide.ID = fmt.Sprintf("slide-ai-%d", len(currentDeck.Slides)+1)
	if slide.BgColor == "" {
		slide.BgColor = "#FFFFFF"
	}
	for i := range slide.Elements {
		slide.Elements[i].ID = fmt.Sprintf("el-ai-%d", i+1)
	}

	return &slide, nil
}
