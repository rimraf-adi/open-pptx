package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"open-pptx/internal/engine"
)

const systemPrompt = `You are an expert AI presentation co-pilot and designer for open-pptx.
Your job is to generate visually stunning, modern, clean presentation slides, shapes, text elements, and code blocks based on user requests.

SLIDE CANVAS SPECIFICATIONS:
- Widescreen 16:9 format: Width 960px, Height 540px.
- All element positions must be within X: 0..960, Y: 0..540.
- Keep text elements wide enough (W: 400..800) to avoid awkward text wrapping.

DESIGN SYSTEM & COLOR PALETTE (Modern Light Theme):
- Slide Background: #FFFFFF (clean white) or #F8FAFC
- Card / Surface Shapes: #F1F5F9 or #E2E8F0
- Primary Accent: #2563EB (electric blue)
- Secondary Accent: #7C3AED (purple)
- Heading text color: #0F172A (dark slate 900)
- Subtitle / Body text color: #334155 (slate 700) or #64748B (slate 500)

ELEMENT TYPES YOU CAN CREATE & MUTATE:
1. Text element: type="text", content="...", position={x, y, w, h}, style={fontSize: 18..52, fontWeight: "bold"|"normal"|"600", color: "#0F172A", textAlign: "left"|"center"|"right"}
2. Shape element: type="shape", shapeType="rect"|"circle", position={x, y, w, h}, style={bgColor: "#2563EB"|"#F1F5F9", borderRadius: 8..16, opacity: 0.1..1.0}
3. Code element: type="code", content="// code...", position={x, y, w, h}, style={fontSize: 14..16, color: "#0F172A", bgColor: "#F1F5F9", borderRadius: 12}

INTENT ACTION MODES:
1. "generate_deck": When user requests creating a presentation / deck / pitch deck.
   JSON structure:
   {
     "action": "generate_deck",
     "title": "Presentation Title",
     "slides": [
       {
         "layout": "title" | "content" | "cards" | "code",
         "bgColor": "#FFFFFF",
         "elements": [ ... ]
       }
     ],
     "message": "Generated a 4-slide presentation about iPhone."
   }

2. "add_slide": When user requests adding a single new slide.
   JSON structure:
   {
     "action": "add_slide",
     "slide": {
       "bgColor": "#FFFFFF",
       "elements": [ ... ]
     },
     "message": "Added a new slide."
   }

3. "edit_slide": When user requests modifying the current slide (e.g. "add a blue card shape", "make title bigger", "add 3 feature cards", "change background to light blue").
   JSON structure:
   {
     "action": "edit_slide",
     "slide": {
       "bgColor": "#FFFFFF",
       "elements": [ ... ]
     },
     "message": "Updated current slide with new shapes and layout."
   }

OUTPUT FORMAT:
You MUST respond ONLY with a valid JSON object matching one of the 3 action structures above. No markdown fences outside the JSON.
`

type Agent struct {
	client *Client
}

func NewAgent(cfg Config) *Agent {
	return &Agent{
		client: NewClient(cfg),
	}
}

type AIResponseEnvelope struct {
	Action  string         `json:"action"` // "generate_deck" | "add_slide" | "edit_slide"
	Title   string         `json:"title"`
	Slides  []engine.Slide `json:"slides"`
	Slide   *engine.Slide  `json:"slide"`
	Message string         `json:"message"`
}

type GeneratedDeckJSON struct {
	Title  string         `json:"title"`
	Slides []engine.Slide `json:"slides"`
}

// ProcessPromptStream handles any prompt (generating decks, adding slides, editing shapes/elements) with SSE streaming.
func (a *Agent) ProcessPromptStream(ctx context.Context, currentDeck *engine.Deck, currentSlideIdx int, prompt string, callback func(chunk StreamChunk)) (*AIResponseEnvelope, error) {
	currentSlideJSON, _ := json.Marshal(currentDeck.Slides[currentSlideIdx])
	deckMetaJSON, _ := json.Marshal(currentDeck.Meta)

	userPrompt := fmt.Sprintf("Deck Meta: %s\nCurrent Slide [%d]: %s\n\nUser Request: '%s'\n\nDetermine action ('generate_deck', 'add_slide', or 'edit_slide') and return JSON envelope.", string(deckMetaJSON), currentSlideIdx+1, string(currentSlideJSON), prompt)

	jsonStr, err := a.client.CompleteStream(ctx, systemPrompt, userPrompt, callback)
	if err != nil {
		return nil, err
	}

	cleanStr := cleanJSONString(jsonStr)

	var resp AIResponseEnvelope
	if err := json.Unmarshal([]byte(cleanStr), &resp); err != nil {
		// Fallback: try parsing as raw deck
		var deckGen GeneratedDeckJSON
		if err2 := json.Unmarshal([]byte(cleanStr), &deckGen); err2 == nil && len(deckGen.Slides) > 0 {
			resp.Action = "generate_deck"
			resp.Title = deckGen.Title
			resp.Slides = deckGen.Slides
		} else {
			return nil, fmt.Errorf("parse AI response: %w\nResponse was: %s", err, jsonStr)
		}
	}

	if resp.Action == "" {
		if len(resp.Slides) > 0 {
			resp.Action = "generate_deck"
		} else if resp.Slide != nil {
			resp.Action = "add_slide"
		} else {
			resp.Action = "generate_deck"
		}
	}

	// Normalize IDs and fallback colors
	if resp.Action == "generate_deck" {
		for sIdx := range resp.Slides {
			resp.Slides[sIdx].ID = fmt.Sprintf("slide-ai-%d", sIdx+1)
			if resp.Slides[sIdx].BgColor == "" {
				resp.Slides[sIdx].BgColor = "#FFFFFF"
			}
			for eIdx := range resp.Slides[sIdx].Elements {
				resp.Slides[sIdx].Elements[eIdx].ID = fmt.Sprintf("el-ai-%d-%d", sIdx+1, eIdx+1)
			}
		}
	} else if resp.Slide != nil {
		if resp.Slide.ID == "" {
			resp.Slide.ID = fmt.Sprintf("slide-ai-%d", len(currentDeck.Slides)+1)
		}
		if resp.Slide.BgColor == "" {
			resp.Slide.BgColor = "#FFFFFF"
		}
		for eIdx := range resp.Slide.Elements {
			if resp.Slide.Elements[eIdx].ID == "" {
				resp.Slide.Elements[eIdx].ID = fmt.Sprintf("el-ai-%d", eIdx+1)
			}
		}
	}

	return &resp, nil
}

// GenerateDeckStream legacy wrapper
func (a *Agent) GenerateDeckStream(ctx context.Context, prompt string, callback func(chunk StreamChunk)) (*engine.Deck, error) {
	env, err := a.ProcessPromptStream(ctx, &engine.Deck{Slides: []engine.Slide{{}}}, 0, prompt, callback)
	if err != nil {
		return nil, err
	}
	deck := engine.NewDeck()
	if env.Title != "" {
		deck.Meta.Title = env.Title
	} else {
		deck.Meta.Title = prompt
	}
	if len(env.Slides) > 0 {
		deck.Slides = env.Slides
	}
	return &deck, nil
}

// AddSlideStream legacy wrapper
func (a *Agent) AddSlideStream(ctx context.Context, currentDeck *engine.Deck, prompt string, callback func(chunk StreamChunk)) (*engine.Slide, error) {
	env, err := a.ProcessPromptStream(ctx, currentDeck, 0, prompt, callback)
	if err != nil {
		return nil, err
	}
	if env.Slide != nil {
		return env.Slide, nil
	}
	if len(env.Slides) > 0 {
		return &env.Slides[0], nil
	}
	return nil, fmt.Errorf("no slide returned by AI")
}

func cleanJSONString(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) > 2 {
			s = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	s = strings.TrimSpace(s)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
