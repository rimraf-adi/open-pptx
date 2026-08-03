package main

import (
	"context"
	"fmt"

	"open-pptx/internal/ai"
	"open-pptx/internal/engine"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the main application struct bound to the Wails frontend.
type App struct {
	ctx      context.Context
	deck     engine.Deck
	history  *engine.History
	filePath string
	aiAgent  *ai.Agent
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{
		deck:    engine.NewDeck(),
		history: engine.NewHistory(100),
		aiAgent: ai.NewAgent(ai.Config{Provider: "nim", Model: "meta/llama-3.3-70b-instruct"}),
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.history.Push(a.deck)
}

// --- Deck Operations ---

// GetDeck returns the current deck.
func (a *App) GetDeck() engine.Deck {
	return a.deck
}

// NewDeck creates a fresh deck, replacing the current one.
func (a *App) NewDeck() engine.Deck {
	a.deck = engine.NewDeck()
	a.filePath = ""
	a.history = engine.NewHistory(100)
	a.history.Push(a.deck)
	return a.deck
}

// UpdateDeckMeta updates the deck's title and author.
func (a *App) UpdateDeckMeta(title, author string) engine.Deck {
	a.deck.Meta.Title = title
	a.deck.Meta.Author = author
	a.pushHistory()
	return a.deck
}

// SetTheme sets the deck's theme.
func (a *App) SetTheme(theme engine.Theme) engine.Deck {
	a.deck.Theme = theme
	a.pushHistory()
	return a.deck
}

// --- Slide Operations ---

// AddSlide adds a new blank slide at the given index (-1 for end).
func (a *App) AddSlide(index int) engine.Deck {
	slide := engine.NewSlide()
	if index < 0 || index >= len(a.deck.Slides) {
		a.deck.Slides = append(a.deck.Slides, slide)
	} else {
		a.deck.Slides = append(a.deck.Slides[:index+1], a.deck.Slides[index:]...)
		a.deck.Slides[index] = slide
	}
	a.pushHistory()
	return a.deck
}

// DeleteSlide removes the slide at the given index.
func (a *App) DeleteSlide(index int) engine.Deck {
	if index < 0 || index >= len(a.deck.Slides) {
		return a.deck
	}
	// Don't delete the last slide
	if len(a.deck.Slides) <= 1 {
		return a.deck
	}
	a.deck.Slides = append(a.deck.Slides[:index], a.deck.Slides[index+1:]...)
	a.pushHistory()
	return a.deck
}

// DuplicateSlide duplicates the slide at the given index.
func (a *App) DuplicateSlide(index int) engine.Deck {
	if index < 0 || index >= len(a.deck.Slides) {
		return a.deck
	}
	original := a.deck.Slides[index]
	// Deep copy elements
	newSlide := engine.Slide{
		ID:       fmt.Sprintf("slide-%d-%d", index, len(a.deck.Slides)),
		Layout:   original.Layout,
		Elements: make([]engine.Element, len(original.Elements)),
		Notes:    original.Notes,
		BgColor:  original.BgColor,
	}
	copy(newSlide.Elements, original.Elements)
	for i := range newSlide.Elements {
		newSlide.Elements[i].ID = fmt.Sprintf("el-dup-%d-%d", i, len(a.deck.Slides))
	}
	// Insert after the original
	after := index + 1
	a.deck.Slides = append(a.deck.Slides[:after], append([]engine.Slide{newSlide}, a.deck.Slides[after:]...)...)
	a.pushHistory()
	return a.deck
}

// ReorderSlide moves a slide from oldIndex to newIndex.
func (a *App) ReorderSlide(oldIndex, newIndex int) engine.Deck {
	if oldIndex < 0 || oldIndex >= len(a.deck.Slides) || newIndex < 0 || newIndex >= len(a.deck.Slides) {
		return a.deck
	}
	slide := a.deck.Slides[oldIndex]
	a.deck.Slides = append(a.deck.Slides[:oldIndex], a.deck.Slides[oldIndex+1:]...)
	a.deck.Slides = append(a.deck.Slides[:newIndex], append([]engine.Slide{slide}, a.deck.Slides[newIndex:]...)...)
	a.pushHistory()
	return a.deck
}

// UpdateSlideBg updates the background color of a slide.
func (a *App) UpdateSlideBg(slideIndex int, bgColor string) engine.Deck {
	if slideIndex < 0 || slideIndex >= len(a.deck.Slides) {
		return a.deck
	}
	a.deck.Slides[slideIndex].BgColor = bgColor
	a.pushHistory()
	return a.deck
}

// --- Element Operations ---

// AddElement adds a new element to the specified slide.
func (a *App) AddElement(slideIndex int, elementType string) engine.Deck {
	if slideIndex < 0 || slideIndex >= len(a.deck.Slides) {
		return a.deck
	}
	el := engine.NewElement(elementType)
	// Set z-index to top
	maxZ := 0
	for _, existing := range a.deck.Slides[slideIndex].Elements {
		if existing.ZIndex > maxZ {
			maxZ = existing.ZIndex
		}
	}
	el.ZIndex = maxZ + 1
	a.deck.Slides[slideIndex].Elements = append(a.deck.Slides[slideIndex].Elements, el)
	a.pushHistory()
	return a.deck
}

// UpdateElement updates an element's properties on the specified slide.
func (a *App) UpdateElement(slideIndex int, element engine.Element) engine.Deck {
	if slideIndex < 0 || slideIndex >= len(a.deck.Slides) {
		return a.deck
	}
	for i, el := range a.deck.Slides[slideIndex].Elements {
		if el.ID == element.ID {
			a.deck.Slides[slideIndex].Elements[i] = element
			break
		}
	}
	a.pushHistory()
	return a.deck
}

// DeleteElement removes an element from the specified slide.
func (a *App) DeleteElement(slideIndex int, elementID string) engine.Deck {
	if slideIndex < 0 || slideIndex >= len(a.deck.Slides) {
		return a.deck
	}
	elements := a.deck.Slides[slideIndex].Elements
	for i, el := range elements {
		if el.ID == elementID {
			a.deck.Slides[slideIndex].Elements = append(elements[:i], elements[i+1:]...)
			break
		}
	}
	a.pushHistory()
	return a.deck
}

// --- Undo / Redo ---

// Undo reverts to the previous state.
func (a *App) Undo() engine.Deck {
	if d := a.history.Undo(); d != nil {
		a.deck = *d
	}
	return a.deck
}

// Redo re-applies the next state.
func (a *App) Redo() engine.Deck {
	if d := a.history.Redo(); d != nil {
		a.deck = *d
	}
	return a.deck
}

// CanUndo returns whether undo is available.
func (a *App) CanUndo() bool {
	return a.history.CanUndo()
}

// CanRedo returns whether redo is available.
func (a *App) CanRedo() bool {
	return a.history.CanRedo()
}

// --- File Operations ---

// SaveFile saves the deck to the current file path, or prompts Save As.
func (a *App) SaveFile() (string, error) {
	if a.filePath == "" {
		return a.SaveFileAs()
	}
	if err := engine.SaveDeck(&a.deck, a.filePath); err != nil {
		return "", err
	}
	return a.filePath, nil
}

// SaveFileAs opens a native Save dialog and saves the deck.
func (a *App) SaveFileAs() (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Presentation",
		DefaultFilename: a.deck.Meta.Title + ".opptx",
		Filters: []runtime.FileFilter{
			{DisplayName: "Open PPTX Files (*.opptx)", Pattern: "*.opptx"},
		},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil // User cancelled
	}
	a.filePath = path
	if err := engine.SaveDeck(&a.deck, a.filePath); err != nil {
		return "", err
	}
	return a.filePath, nil
}

// OpenFile opens a native file dialog and loads a deck.
func (a *App) OpenFile() (engine.Deck, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open Presentation",
		Filters: []runtime.FileFilter{
			{DisplayName: "Open PPTX Files (*.opptx)", Pattern: "*.opptx"},
			{DisplayName: "PowerPoint Files (*.pptx)", Pattern: "*.pptx"},
		},
	})
	if err != nil {
		return a.deck, err
	}
	if path == "" {
		return a.deck, nil // User cancelled
	}
	d, err := engine.LoadDeck(path)
	if err != nil {
		return a.deck, err
	}
	a.deck = *d
	a.filePath = path
	a.history = engine.NewHistory(100)
	a.history.Push(a.deck)
	return a.deck, nil
}

// GetFilePath returns the current file path.
func (a *App) GetFilePath() string {
	return a.filePath
}

// --- AI Operations ---

// GenerateDeckWithAI creates a new presentation using the AI Agent.
func (a *App) GenerateDeckWithAI(prompt string) (engine.Deck, error) {
	deck, err := a.aiAgent.GenerateDeckStream(a.ctx, prompt, func(chunk ai.StreamChunk) {
		runtime.EventsEmit(a.ctx, "ai-stream-chunk", chunk)
	})
	if err != nil {
		return a.deck, err
	}
	a.deck = *deck
	a.filePath = ""
	a.history = engine.NewHistory(100)
	a.history.Push(a.deck)
	return a.deck, nil
}

// AddSlideWithAI generates a new slide and appends it to the deck using AI.
func (a *App) AddSlideWithAI(prompt string) (engine.Deck, error) {
	slide, err := a.aiAgent.AddSlideStream(a.ctx, &a.deck, prompt, func(chunk ai.StreamChunk) {
		runtime.EventsEmit(a.ctx, "ai-stream-chunk", chunk)
	})
	if err != nil {
		return a.deck, err
	}
	a.deck.Slides = append(a.deck.Slides, *slide)
	a.pushHistory()
	return a.deck, nil
}

// --- Internal ---

func (a *App) pushHistory() {
	a.history.Push(a.deck)
}

