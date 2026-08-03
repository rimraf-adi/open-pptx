// Package engine provides the core slide data model and operations for open-pptx.
package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// --- Data Model ---

// Position represents an element's position and dimensions on a slide.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// Style represents visual styling for an element.
type Style struct {
	FontSize    int     `json:"fontSize,omitempty"`
	FontWeight  string  `json:"fontWeight,omitempty"`
	FontFamily  string  `json:"fontFamily,omitempty"`
	Color       string  `json:"color,omitempty"`
	BgColor     string  `json:"bgColor,omitempty"`
	BorderColor string  `json:"borderColor,omitempty"`
	BorderWidth float64 `json:"borderWidth,omitempty"`
	BorderRadius float64 `json:"borderRadius,omitempty"`
	Opacity     float64 `json:"opacity,omitempty"`
	TextAlign   string  `json:"textAlign,omitempty"`
	LineHeight  float64 `json:"lineHeight,omitempty"`
}

// Element represents a single element on a slide (text, image, shape, code).
type Element struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"` // "text", "image", "shape", "code"
	Content  string   `json:"content"`
	Position Position `json:"position"`
	Style    Style    `json:"style"`
	ZIndex   int      `json:"zIndex"`
	// Shape-specific
	ShapeType string `json:"shapeType,omitempty"` // "rect", "circle", "triangle", "line"
	// Image-specific
	ImageURL string `json:"imageUrl,omitempty"`
}

// Slide represents a single slide in a deck.
type Slide struct {
	ID       string    `json:"id"`
	Layout   string    `json:"layout,omitempty"` // "blank", "title", "content", "two-column"
	Elements []Element `json:"elements"`
	Notes    string    `json:"notes,omitempty"`
	BgColor  string    `json:"bgColor,omitempty"`
}

// ThemeColors defines the color palette for a deck theme.
type ThemeColors struct {
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
	Bg        string `json:"bg"`
	Surface   string `json:"surface"`
	Text      string `json:"text"`
	TextMuted string `json:"textMuted"`
}

// ThemeFonts defines the font pairing for a deck theme.
type ThemeFonts struct {
	Heading string `json:"heading"`
	Body    string `json:"body"`
}

// Theme defines the visual theme for a deck.
type Theme struct {
	Name   string      `json:"name"`
	Colors ThemeColors `json:"colors"`
	Fonts  ThemeFonts  `json:"fonts"`
}

// DeckMeta holds metadata about a deck.
type DeckMeta struct {
	Title    string `json:"title"`
	Author   string `json:"author"`
	Created  string `json:"created"`
	Modified string `json:"modified"`
}

// Deck is the top-level presentation data structure.
type Deck struct {
	Version string   `json:"version"`
	Meta    DeckMeta `json:"meta"`
	Theme   Theme    `json:"theme"`
	Slides  []Slide  `json:"slides"`
}

// --- History (Undo/Redo) ---

type historyEntry struct {
	deck Deck
}

// History manages undo/redo state.
type History struct {
	entries []historyEntry
	index   int
	maxSize int
}

// NewHistory creates a new history manager.
func NewHistory(maxSize int) *History {
	return &History{
		entries: make([]historyEntry, 0, maxSize),
		index:   -1,
		maxSize: maxSize,
	}
}

// Push saves a deck snapshot to history.
func (h *History) Push(d Deck) {
	// Discard any future entries (redo stack) beyond current index
	if h.index < len(h.entries)-1 {
		h.entries = h.entries[:h.index+1]
	}
	// Deep copy via JSON to avoid mutation issues
	data, _ := json.Marshal(d)
	var copy Deck
	json.Unmarshal(data, &copy)

	h.entries = append(h.entries, historyEntry{deck: copy})
	if len(h.entries) > h.maxSize {
		h.entries = h.entries[1:]
	}
	h.index = len(h.entries) - 1
}

// Undo returns the previous state, or nil if at the beginning.
func (h *History) Undo() *Deck {
	if h.index <= 0 {
		return nil
	}
	h.index--
	d := h.entries[h.index].deck
	return &d
}

// Redo returns the next state, or nil if at the end.
func (h *History) Redo() *Deck {
	if h.index >= len(h.entries)-1 {
		return nil
	}
	h.index++
	d := h.entries[h.index].deck
	return &d
}

// CanUndo returns whether undo is possible.
func (h *History) CanUndo() bool {
	return h.index > 0
}

// CanRedo returns whether redo is possible.
func (h *History) CanRedo() bool {
	return h.index < len(h.entries)-1
}

// --- Default Theme ---

// DefaultTheme returns the default light theme.
func DefaultTheme() Theme {
	return Theme{
		Name: "Canvas Light",
		Colors: ThemeColors{
			Primary:   "#2563EB",
			Secondary: "#7C3AED",
			Bg:        "#F8FAFC",
			Surface:   "#FFFFFF",
			Text:      "#0F172A",
			TextMuted: "#64748B",
		},
		Fonts: ThemeFonts{
			Heading: "Inter",
			Body:    "Inter",
		},
	}
}

// --- Factory Functions ---

// NewDeck creates a new empty deck with a default title slide.
func NewDeck() Deck {
	now := time.Now().UTC().Format(time.RFC3339)
	return Deck{
		Version: "1.0",
		Meta: DeckMeta{
			Title:    "Untitled Presentation",
			Author:   "",
			Created:  now,
			Modified: now,
		},
		Theme: DefaultTheme(),
		Slides: []Slide{
			{
				ID:      generateID("slide"),
				Layout:  "title",
				BgColor: "#FFFFFF",
				Elements: []Element{
					{
						ID:      generateID("el"),
						Type:    "text",
						Content: "Untitled Presentation",
						Position: Position{
							X: 80, Y: 200, W: 800, H: 100,
						},
						Style: Style{
							FontSize:   52,
							FontWeight: "bold",
							Color:      "#0F172A",
							TextAlign:  "center",
							FontFamily: "Inter",
						},
						ZIndex: 1,
					},
					{
						ID:      generateID("el"),
						Type:    "text",
						Content: "Click to add subtitle",
						Position: Position{
							X: 200, Y: 320, W: 560, H: 50,
						},
						Style: Style{
							FontSize:   24,
							FontWeight: "normal",
							Color:      "#64748B",
							TextAlign:  "center",
							FontFamily: "Inter",
						},
						ZIndex: 2,
					},
				},
			},
		},
	}
}

// NewSlide creates a new blank slide.
func NewSlide() Slide {
	return Slide{
		ID:       generateID("slide"),
		Layout:   "blank",
		Elements: []Element{},
		BgColor:  "#FFFFFF",
	}
}

// NewElement creates a new element of the given type with default properties.
func NewElement(elementType string) Element {
	el := Element{
		ID:     generateID("el"),
		Type:   elementType,
		ZIndex: 1,
	}
	switch elementType {
	case "text":
		el.Content = "New text"
		el.Position = Position{X: 100, Y: 100, W: 300, H: 60}
		el.Style = Style{
			FontSize:   24,
			FontWeight: "normal",
			Color:      "#0F172A",
			TextAlign:  "left",
			FontFamily: "Inter",
		}
	case "shape":
		return NewShapeElement("rect")
	case "image":
		el.Position = Position{X: 100, Y: 100, W: 300, H: 200}
		el.Content = ""
	case "code":
		el.Content = "// your code here"
		el.Position = Position{X: 80, Y: 100, W: 800, H: 300}
		el.Style = Style{
			FontSize:     16,
			FontFamily:   "JetBrains Mono, monospace",
			Color:        "#0F172A",
			BgColor:      "#F1F5F9",
			BorderRadius: 12,
		}
	}
	return el
}

// NewShapeElement creates a shape element of a specific type (rect, circle, triangle, star, diamond, arrow, line).
func NewShapeElement(shapeType string) Element {
	if shapeType == "" {
		shapeType = "rect"
	}
	w, h := 200.0, 150.0
	if shapeType == "circle" || shapeType == "star" || shapeType == "diamond" {
		w, h = 160.0, 160.0
	} else if shapeType == "line" {
		w, h = 300.0, 6.0
	} else if shapeType == "arrow" || shapeType == "triangle" {
		w, h = 180.0, 120.0
	}

	bgColor := "#2563EB"
	if shapeType == "star" || shapeType == "diamond" {
		bgColor = "#7C3AED"
	} else if shapeType == "triangle" {
		bgColor = "#059669"
	} else if shapeType == "arrow" {
		bgColor = "#D97706"
	}

	return Element{
		ID:        generateID("el"),
		Type:      "shape",
		ShapeType: shapeType,
		Position:  Position{X: 150, Y: 150, W: w, H: h},
		Style: Style{
			BgColor:      bgColor,
			BorderRadius: 8,
			Opacity:      1,
		},
		ZIndex: 1,
	}
}

// --- Persistence ---

// SaveDeck saves a deck to a .opptx file (JSON).
func SaveDeck(d *Deck, path string) error {
	d.Meta.Modified = time.Now().UTC().Format(time.RFC3339)

	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal deck: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	AddRecentDeck(path, d.Meta.Title)
	return nil
}

// LoadDeck loads a deck from a .opptx file (JSON).
func LoadDeck(path string) (*Deck, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var d Deck
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("unmarshal deck: %w", err)
	}

	AddRecentDeck(path, d.Meta.Title)
	return &d, nil
}

// --- Recent Files ---

type RecentDeckItem struct {
	Path     string `json:"path"`
	Title    string `json:"title"`
	Modified string `json:"modified"`
}

func GetRecentDecks() []RecentDeckItem {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	recentsPath := filepath.Join(home, ".open-pptx-recents.json")
	data, err := os.ReadFile(recentsPath)
	if err != nil {
		return nil
	}
	var items []RecentDeckItem
	_ = json.Unmarshal(data, &items)
	return items
}

func AddRecentDeck(path, title string) {
	if path == "" {
		return
	}
	if title == "" {
		title = filepath.Base(path)
	}
	items := GetRecentDecks()
	var updated []RecentDeckItem
	newItem := RecentDeckItem{
		Path:     path,
		Title:    title,
		Modified: time.Now().Format("2006-01-02 15:04"),
	}
	updated = append(updated, newItem)
	for _, item := range items {
		if item.Path != path && len(updated) < 10 {
			updated = append(updated, item)
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	recentsPath := filepath.Join(home, ".open-pptx-recents.json")
	data, _ := json.MarshalIndent(updated, "", "  ")
	_ = os.WriteFile(recentsPath, data, 0644)
}

// --- ID Generation ---

var idCounter int

func generateID(prefix string) string {
	idCounter++
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixMilli(), idCounter)
}
