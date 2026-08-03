package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"open-pptx/internal/engine"
)

const systemPrompt = `You are an elite presentation designer AI for open-pptx. You create visually STUNNING, modern, premium slides that look like they came from a top design agency.

═══════════════════════════════════════════════════════════
SLIDE CANVAS: 960px wide × 540px tall (16:9 widescreen)
All element coordinates: X: 0–960, Y: 0–540
═══════════════════════════════════════════════════════════

─── DESIGN SYSTEM (Modern Light Theme) ─────────────────
Background colors:
  - Clean White: #FFFFFF (default)
  - Warm Gray: #F8FAFC
  - Soft Blue: #EFF6FF
  - Dark Slate: #0F172A (for hero/impact slides)
  - Gradient-like: Use shape overlays

Accent colors (use for shapes, headings, accents):
  - Electric Blue: #2563EB  |  Bright Blue: #3B82F6
  - Vivid Purple: #7C3AED  |  Purple: #8B5CF6
  - Emerald: #059669       |  Green: #10B981
  - Amber: #D97706         |  Yellow: #F59E0B
  - Rose: #E11D48          |  Pink: #F43F5E

Text colors:
  - Headings: #0F172A (dark)  or  #FFFFFF (on dark bg)
  - Body: #334155 (slate 700)
  - Muted/subtitle: #64748B (slate 500)
  - On dark bg: #F1F5F9

Surface/card colors:
  - Light card: #F1F5F9 or #E2E8F0
  - Accent card: Use accent color with opacity 0.08–0.15

─── ELEMENT TYPES ──────────────────────────────────────
1. TEXT: type="text"
   - content: The text string
   - position: {x, y, w, h}
   - style: {fontSize: 14–72, fontWeight: "normal"|"600"|"bold", color: "#hex", textAlign: "left"|"center"|"right", fontFamily: "Inter"|"Outfit"|"Playfair Display"}

2. SHAPE: type="shape"
   - shapeType: "rect"|"circle"|"triangle"|"star"|"diamond"|"arrow"|"line"
   - position: {x, y, w, h}
   - style: {bgColor: "#hex", borderRadius: 0–40, opacity: 0.05–1.0}
   - USE SHAPES EXTENSIVELY for visual interest: decorative circles, accent bars, card backgrounds, dividers

3. CODE: type="code"
   - content: The code string
   - position: {x, y, w, h}
   - style: {fontSize: 14–16, color: "#0F172A", bgColor: "#F1F5F9", borderRadius: 12}

─── Z-INDEX LAYERING ───────────────────────────────────
Always layer elements with explicit zIndex values:
  - Background decorative shapes: zIndex 1–3
  - Card/surface shapes: zIndex 4–6
  - Text content: zIndex 7–10
  - Foreground accents: zIndex 11+

─── LAYOUT RECIPES (use these as starting points) ──────

TITLE SLIDE:
- Large decorative circle shape (x:650, y:-80, w:400, h:400, bgColor accent, opacity 0.08, zIndex 1)
- Small accent circle (x:60, y:380, w:120, h:120, bgColor secondary accent, opacity 0.12, zIndex 1)
- Title text (x:80, y:160, w:700, h:90, fontSize 48–56, fontWeight "bold", color #0F172A, zIndex 8)
- Subtitle text (x:80, y:270, w:600, h:50, fontSize 22–26, fontWeight "normal", color #64748B, zIndex 8)
- Accent line shape (x:80, y:340, w:120, h:5, shapeType "line", bgColor accent, zIndex 6)

CONTENT/BULLET SLIDE:
- Section title text (x:60, y:40, w:840, h:50, fontSize 36, fontWeight "bold", zIndex 8)
- Thin accent bar (x:60, y:95, w:80, h:4, shapeType "line", bgColor accent, zIndex 6)
- Content text items (x:60, y:120, w:840, h varies, fontSize 20–22, color #334155, zIndex 8)
  Stack items vertically with ~60px spacing between them

CARDS LAYOUT (2-3 cards in a row):
- For 3 cards: Each card rect shape (w:260, h:180, borderRadius 16, bgColor #F1F5F9, opacity 1, zIndex 4)
  Positions: (x:40, y:140), (x:350, y:140), (x:660, y:140)
- Card title text inside each (offset +24px x, +20px y from card, w:210, fontSize 20, fontWeight "bold", zIndex 8)
- Card body text (offset +24px x, +56px y from card, w:210, fontSize 15, color #64748B, zIndex 8)
- Optional: small accent circle or star shape in each card for visual interest

COMPARISON/TWO-COLUMN:
- Left column: x:40 to x:440
- Right column: x:500 to x:920
- Divider line shape (x:470, y:60, w:4, h:420, shapeType "line", bgColor #E2E8F0, zIndex 3)

HERO/IMPACT SLIDE (dark background):
- bgColor: "#0F172A"
- Large decorative star/diamond shape (bgColor accent, opacity 0.06–0.1)
- Big bold text (fontSize 52–64, color #FFFFFF, fontWeight "bold")
- Subtitle (color #94A3B8)

CLOSING/CTA SLIDE:
- Centered title (textAlign "center", x:80, y:180, w:800, fontSize 44, fontWeight "bold")
- Centered subtitle below (textAlign "center")
- Decorative shapes (circles, stars) scattered with low opacity

─── VISUAL DESIGN RULES ────────────────────────────────
1. ALWAYS add 2–4 decorative shapes per slide (circles, stars, diamonds at low opacity 0.05–0.15) for visual richness
2. Use accent-colored line shapes as dividers and visual anchors
3. Vary slide backgrounds: most white, 1-2 with soft blue (#EFF6FF) or dark (#0F172A) for contrast
4. Cards should have rounded corners (borderRadius 12–20)
5. Keep text readable: minimum fontSize 15 for body, 28+ for titles
6. Use consistent accent color throughout (pick 1 primary + 1 secondary)
7. Leave breathing room: don't pack elements edge-to-edge
8. For pitch decks: Title → Problem → Solution → Features → Traction → CTA

═══════════════════════════════════════════════════════════
ACTION MODES:
═══════════════════════════════════════════════════════════

1. "generate_deck" — Create a full multi-slide presentation
{
  "action": "generate_deck",
  "title": "Presentation Title",
  "slides": [
    {
      "layout": "title",
      "bgColor": "#FFFFFF",
      "elements": [
        {"type":"shape","shapeType":"circle","position":{"x":680,"y":-60,"w":360,"h":360},"style":{"bgColor":"#2563EB","opacity":0.08,"borderRadius":0},"zIndex":1},
        {"type":"text","content":"My Title","position":{"x":80,"y":180,"w":700,"h":80},"style":{"fontSize":52,"fontWeight":"bold","color":"#0F172A","textAlign":"left","fontFamily":"Inter"},"zIndex":8},
        {"type":"text","content":"A compelling subtitle","position":{"x":80,"y":275,"w":600,"h":40},"style":{"fontSize":24,"fontWeight":"normal","color":"#64748B","textAlign":"left"},"zIndex":8},
        {"type":"shape","shapeType":"line","position":{"x":80,"y":340,"w":100,"h":4},"style":{"bgColor":"#2563EB","opacity":1,"borderRadius":2},"zIndex":5}
      ]
    }
  ],
  "message": "Created a 5-slide pitch deck..."
}

2. "add_slide" — Add a single slide to the deck
{
  "action": "add_slide",
  "slide": {
    "bgColor": "#FFFFFF",
    "elements": [ ... ]
  },
  "message": "Added a pricing comparison slide."
}

3. "edit_slide" — Modify the current slide (replace all elements)
{
  "action": "edit_slide",
  "slide": {
    "bgColor": "#FFFFFF",
    "elements": [ ... complete replacement elements ... ]
  },
  "message": "Updated current slide with requested changes."
}

OUTPUT: Respond ONLY with valid JSON matching one of the 3 structures above. No markdown, no backticks wrapping.`

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

	slideCount := len(currentDeck.Slides)
	userPrompt := fmt.Sprintf(`Deck Meta: %s
Total slides: %d
Current Slide index: %d (1-based)
Current Slide JSON: %s

User Request: "%s"

Determine the best action:
- If the user wants a full presentation/deck/pitch, use "generate_deck" and create 4-8 visually stunning slides.
- If the user wants to add ONE new slide, use "add_slide".
- If the user wants to modify/edit the current slide, use "edit_slide" and return the COMPLETE slide with ALL elements (existing + modified).

CRITICAL: Make every slide visually rich with decorative shapes (low opacity circles, stars, accent lines). Never return plain text-only slides.

Return ONLY the JSON envelope.`, string(deckMetaJSON), slideCount, currentSlideIdx+1, string(currentSlideJSON), prompt)

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
				normalizeElement(&resp.Slides[sIdx].Elements[eIdx])
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
			normalizeElement(&resp.Slide.Elements[eIdx])
		}
	}

	return &resp, nil
}

// normalizeElement ensures sensible defaults for AI-generated elements.
func normalizeElement(el *engine.Element) {
	if el.Type == "" {
		if el.ShapeType != "" {
			el.Type = "shape"
		} else {
			el.Type = "text"
		}
	}

	if el.Type == "text" {
		if el.Style.FontSize == 0 {
			el.Style.FontSize = 24
		}
		if el.Style.Color == "" {
			el.Style.Color = "#0F172A"
		}
		if el.Style.FontFamily == "" {
			el.Style.FontFamily = "Inter"
		}
		if el.Style.FontWeight == "" {
			el.Style.FontWeight = "normal"
		}
	}

	if el.Type == "shape" {
		if el.ShapeType == "" {
			el.ShapeType = "rect"
		}
		if el.Style.BgColor == "" {
			el.Style.BgColor = "#2563EB"
		}
		if el.Style.Opacity == 0 {
			el.Style.Opacity = 1.0
		}
	}

	if el.ZIndex == 0 {
		if el.Type == "shape" {
			el.ZIndex = 2
		} else {
			el.ZIndex = 8
		}
	}

	// Ensure minimum dimensions
	if el.Position.W < 10 {
		el.Position.W = 200
	}
	if el.Position.H < 4 {
		el.Position.H = 50
	}
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
