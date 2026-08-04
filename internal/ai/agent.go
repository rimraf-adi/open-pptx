package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"open-pptx/internal/engine"
)

const systemPrompt = `You are a world-class presentation designer. You create clean, professional, well-structured slide decks.

CANVAS: 960px wide × 540px tall (16:9). All positions must stay within these bounds.

══════════════════════════════════════════
STRICT RULES — FOLLOW EVERY ONE:
══════════════════════════════════════════

1. NO random decorative shapes. No floating circles, stars, or diamonds in backgrounds. Every element must serve a purpose.
2. Use ONLY purposeful shapes: card backgrounds for grouping content, accent lines as dividers, or icon-like shapes that relate to content.
3. All text must be fully visible — never overlapping, never cut off. Leave 40px+ margins from slide edges.
4. Elements must NEVER overlap unless intentionally layered (e.g., text ON TOP of a card shape).
5. When placing cards in a row, calculate positions precisely so they don't overlap. Include 20-30px gaps between cards.
6. Keep slides clean and uncluttered. Fewer well-placed elements beats many scattered ones.
7. Body text fontSize should be 18-22px for readability. Titles 32-48px. Never below 16px.

══════════════════════════════════════════
DESIGN TOKENS:
══════════════════════════════════════════

Backgrounds: #FFFFFF (default), #F8FAFC (warm), #0F172A (dark hero)
Headings: #0F172A (on light bg) or #FFFFFF (on dark bg)
Body text: #334155 (on light) or #CBD5E1 (on dark)
Muted text: #64748B
Accent blue: #2563EB   |   Purple: #7C3AED   |   Emerald: #059669
Amber: #D97706   |   Rose: #E11D48
Card surface: #F1F5F9 (light gray card)

══════════════════════════════════════════
ELEMENT TYPES:
══════════════════════════════════════════

TEXT: {"type":"text","content":"...","position":{"x":N,"y":N,"w":N,"h":N},"style":{"fontSize":N,"fontWeight":"normal"|"600"|"bold","color":"#hex","textAlign":"left"|"center"|"right","fontFamily":"Inter"},"zIndex":N}

SHAPE: {"type":"shape","shapeType":"rect"|"circle"|"triangle"|"star"|"diamond"|"arrow"|"line","position":{"x":N,"y":N,"w":N,"h":N},"style":{"bgColor":"#hex","borderRadius":N,"opacity":N},"zIndex":N}
  - Use shapes ONLY as: card backgrounds (rect with borderRadius 12-16), accent lines (line, h=3-4), or meaningful icons.

CODE: {"type":"code","content":"...","position":{"x":N,"y":N,"w":N,"h":N},"style":{"fontSize":14,"color":"#334155","bgColor":"#F1F5F9","borderRadius":12},"zIndex":N}

══════════════════════════════════════════
EXACT LAYOUT TEMPLATES (use these coordinates):
══════════════════════════════════════════

--- TITLE SLIDE ---
Title: {x:80, y:180, w:800, h:70, fontSize:48, fontWeight:"bold", textAlign:"center", color:"#0F172A", zIndex:5}
Subtitle: {x:160, y:270, w:640, h:40, fontSize:22, fontWeight:"normal", textAlign:"center", color:"#64748B", zIndex:5}
Accent line: {type:"shape", shapeType:"line", x:400, y:330, w:160, h:3, bgColor:"#2563EB", zIndex:4}

--- SECTION HEADER SLIDE ---
Section title: {x:80, y:200, w:800, h:60, fontSize:40, fontWeight:"bold", textAlign:"center", color:"#0F172A", zIndex:5}
Section subtitle: {x:160, y:275, w:640, h:40, fontSize:20, color:"#64748B", textAlign:"center", zIndex:5}

--- BULLET POINTS SLIDE ---
Slide title: {x:60, y:40, w:840, h:50, fontSize:32, fontWeight:"bold", color:"#0F172A", zIndex:5}
Accent line: {type:"shape", shapeType:"line", x:60, y:95, w:80, h:3, bgColor:"#2563EB", zIndex:4}
Bullet 1: {x:60, y:120, w:840, h:36, fontSize:20, color:"#334155", zIndex:5, content:"• Point one"}
Bullet 2: {x:60, y:168, w:840, h:36, fontSize:20, color:"#334155", zIndex:5, content:"• Point two"}
Bullet 3: {x:60, y:216, w:840, h:36, fontSize:20, color:"#334155", zIndex:5, content:"• Point three"}
Bullet 4: {x:60, y:264, w:840, h:36, fontSize:20, color:"#334155", zIndex:5, content:"• Point four"}
(Stack bullets with 48px vertical spacing. Max 6-7 bullets per slide.)

--- 3-CARD ROW LAYOUT ---
Slide title: {x:60, y:30, w:840, h:45, fontSize:30, fontWeight:"bold", color:"#0F172A", zIndex:5}
Card 1 bg: {type:"shape", shapeType:"rect", x:40, y:100, w:270, h:200, bgColor:"#F1F5F9", borderRadius:14, opacity:1, zIndex:2}
Card 1 title: {x:60, y:115, w:230, h:30, fontSize:20, fontWeight:"bold", color:"#0F172A", zIndex:5}
Card 1 body: {x:60, y:155, w:230, h:120, fontSize:16, color:"#64748B", zIndex:5}
Card 2 bg: {type:"shape", shapeType:"rect", x:340, y:100, w:270, h:200, bgColor:"#F1F5F9", borderRadius:14, opacity:1, zIndex:2}
Card 2 title: {x:360, y:115, w:230, h:30, fontSize:20, fontWeight:"bold", color:"#0F172A", zIndex:5}
Card 2 body: {x:360, y:155, w:230, h:120, fontSize:16, color:"#64748B", zIndex:5}
Card 3 bg: {type:"shape", shapeType:"rect", x:640, y:100, w:270, h:200, bgColor:"#F1F5F9", borderRadius:14, opacity:1, zIndex:2}
Card 3 title: {x:660, y:115, w:230, h:30, fontSize:20, fontWeight:"bold", color:"#0F172A", zIndex:5}
Card 3 body: {x:660, y:155, w:230, h:120, fontSize:16, color:"#64748B", zIndex:5}

--- 2-CARD ROW LAYOUT ---
Card 1 bg: {x:40, y:100, w:420, h:220, bgColor:"#F1F5F9", borderRadius:14, zIndex:2}
Card 1 title: {x:65, y:120, w:370, h:30, fontSize:22, fontWeight:"bold", zIndex:5}
Card 1 body: {x:65, y:160, w:370, h:130, fontSize:17, color:"#64748B", zIndex:5}
Card 2 bg: {x:490, y:100, w:420, h:220, bgColor:"#F1F5F9", borderRadius:14, zIndex:2}
Card 2 title: {x:515, y:120, w:370, h:30, fontSize:22, fontWeight:"bold", zIndex:5}
Card 2 body: {x:515, y:160, w:370, h:130, fontSize:17, color:"#64748B", zIndex:5}

--- TWO-COLUMN SPLIT ---
Left title: {x:60, y:60, w:400, h:40, fontSize:26, fontWeight:"bold", zIndex:5}
Left content: {x:60, y:110, w:400, h:360, fontSize:18, color:"#334155", zIndex:5}
Divider: {type:"shape", shapeType:"line", x:490, y:60, w:3, h:400, bgColor:"#E2E8F0", zIndex:3}
Right title: {x:520, y:60, w:400, h:40, fontSize:26, fontWeight:"bold", zIndex:5}
Right content: {x:520, y:110, w:400, h:360, fontSize:18, color:"#334155", zIndex:5}

--- CLOSING/CTA SLIDE ---
Heading: {x:80, y:180, w:800, h:60, fontSize:40, fontWeight:"bold", textAlign:"center", color:"#0F172A", zIndex:5}
Subtext: {x:160, y:260, w:640, h:40, fontSize:20, textAlign:"center", color:"#64748B", zIndex:5}

══════════════════════════════════════════
CHAIN OF THOUGHT — THINK BEFORE GENERATING:
══════════════════════════════════════════

Before generating JSON, mentally plan:
1. What is the topic? What are the key sections/points?
2. How many slides are needed? (Title + 3-6 content slides + closing = 5-8 total)
3. For EACH slide: what layout template fits best? What content goes on it?
4. What is the logical flow? (Title → Overview → Detail sections → Summary/CTA)
5. Are all element positions calculated correctly with no overlaps?

ALWAYS generate a COMPLETE deck when asked. A "pitch deck" or "presentation" means 5-8 slides minimum.

══════════════════════════════════════════
ACTIONS:
══════════════════════════════════════════

1. "generate_deck" — Full multi-slide presentation
{"action":"generate_deck","title":"Title","slides":[{"bgColor":"#FFFFFF","elements":[...]}],"message":"Created a 6-slide deck about X."}

2. "add_slide" — Single new slide
{"action":"add_slide","slide":{"bgColor":"#FFFFFF","elements":[...]},"message":"Added a pricing slide."}

3. "edit_slide" — Modify current slide
{"action":"edit_slide","slide":{"bgColor":"#FFFFFF","elements":[...]},"message":"Updated the slide layout."}

OUTPUT: Return ONLY valid JSON. No markdown. No backticks.`

type Agent struct {
	client *Client
}

func NewAgent(cfg Config) *Agent {
	return &Agent{
		client: NewClient(cfg),
	}
}

type AIResponseEnvelope struct {
	Action  string         `json:"action"`
	Title   string         `json:"title"`
	Slides  []engine.Slide `json:"slides"`
	Slide   *engine.Slide  `json:"slide"`
	Message string         `json:"message"`
}

type GeneratedDeckJSON struct {
	Title  string         `json:"title"`
	Slides []engine.Slide `json:"slides"`
}

// ProcessPromptStream handles any prompt with SSE streaming.
func (a *Agent) ProcessPromptStream(ctx context.Context, currentDeck *engine.Deck, currentSlideIdx int, prompt string, callback func(chunk StreamChunk)) (*AIResponseEnvelope, error) {
	if currentSlideIdx < 0 || currentSlideIdx >= len(currentDeck.Slides) {
		currentSlideIdx = 0
	}

	var slidesOutline []string
	for i, s := range currentDeck.Slides {
		title := fmt.Sprintf("Slide %d", i+1)
		for _, el := range s.Elements {
			if el.Type == "text" && (el.Style.FontSize >= 28 || el.Style.FontWeight == "bold") {
				title = fmt.Sprintf("Slide %d: \"%s\"", i+1, el.Content)
				break
			}
		}
		slidesOutline = append(slidesOutline, title)
	}
	slidesOutlineStr := strings.Join(slidesOutline, "\n")

	currentSlideJSON, _ := json.Marshal(currentDeck.Slides[currentSlideIdx])
	deckMetaJSON, _ := json.Marshal(currentDeck.Meta)

	slideCount := len(currentDeck.Slides)

	// Cap prompt length at 12,000 characters (~3,000 tokens) to prevent TPM limit overflow
	trimmedPrompt := prompt
	if len(trimmedPrompt) > 12000 {
		trimmedPrompt = trimmedPrompt[:12000] + "\n\n[...content trimmed to fit AI rate limits...]"
	}

	userPrompt := fmt.Sprintf(`Deck Meta: %s
Total slides: %d | Targeted Slide Index: %d (1-based)

All Slides Outline Index:
%s

Targeted Slide [%d] JSON: %s

User Request: "%s"

IMPORTANT SLIDE TAGGING & SELECTION RULES:
- If the user prompt mentions @slide1, @slide2, @slideN etc., target that specific slide for "edit_slide".
- If the user asks for a full deck/presentation/pitch, use "generate_deck" with 5-8 well-designed slides.
- If the user wants one new slide, use "add_slide".
- If editing a slide ("edit_slide"), return the complete slide with updated/repositioned elements.
- Double-check that NO elements overlap. Calculate x,y,w,h carefully.
- Every slide must have substantive content — not just a title and one bullet.

Return ONLY the JSON envelope.`, string(deckMetaJSON), slideCount, currentSlideIdx+1, slidesOutlineStr, currentSlideIdx+1, string(currentSlideJSON), trimmedPrompt)

	jsonStr, err := a.client.CompleteStream(ctx, systemPrompt, userPrompt, callback)
	if err != nil {
		return nil, err
	}

	cleanStr := cleanJSONString(jsonStr)

	var resp AIResponseEnvelope
	if err := json.Unmarshal([]byte(cleanStr), &resp); err != nil {
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

	// Normalize
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

// normalizeElement ensures sensible defaults and clamps positions to canvas bounds.
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
			el.Style.FontSize = 20
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
			el.Style.BgColor = "#F1F5F9"
		}
		if el.Style.Opacity == 0 {
			el.Style.Opacity = 1.0
		}
	}

	if el.ZIndex == 0 {
		if el.Type == "shape" {
			el.ZIndex = 2
		} else {
			el.ZIndex = 5
		}
	}

	// Clamp positions to canvas bounds (960x540)
	if el.Position.X < 0 {
		el.Position.X = 0
	}
	if el.Position.Y < 0 {
		el.Position.Y = 0
	}
	if el.Position.W < 10 {
		el.Position.W = 200
	}
	if el.Position.H < 3 {
		el.Position.H = 40
	}
	// Ensure element doesn't extend beyond canvas right/bottom
	if el.Position.X+el.Position.W > 960 {
		el.Position.W = 960 - el.Position.X
		if el.Position.W < 10 {
			el.Position.X = 0
			el.Position.W = 200
		}
	}
	if el.Position.Y+el.Position.H > 540 {
		el.Position.H = 540 - el.Position.Y
		if el.Position.H < 3 {
			el.Position.Y = 0
			el.Position.H = 40
		}
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
