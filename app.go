package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		aiAgent: ai.NewAgent(ai.Config{Provider: "groq", Model: "llama-3.3-70b-versatile"}),
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

// AddShapeElement adds a specific shape type to the specified slide.
func (a *App) AddShapeElement(slideIndex int, shapeType string) engine.Deck {
	if slideIndex < 0 || slideIndex >= len(a.deck.Slides) {
		return a.deck
	}
	el := engine.NewShapeElement(shapeType)
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

// GetRecentDecks returns recent presentation files.
func (a *App) GetRecentDecks() []engine.RecentDeckItem {
	return engine.GetRecentDecks()
}

// OpenFileByPath opens a presentation by path directly.
func (a *App) OpenFileByPath(path string) (engine.Deck, error) {
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

// SelectImageFile opens a file dialog to select an image and returns a base64 data URL.
func (a *App) SelectImageFile() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Image or Clipart",
		Filters: []runtime.FileFilter{
			{DisplayName: "Image Files (*.png, *.jpg, *.jpeg, *.webp, *.svg, *.gif)", Pattern: "*.png;*.jpg;*.jpeg;*.webp;*.svg;*.gif"},
		},
	})
	if err != nil || path == "" {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(path))
	mimeType := "image/png"
	switch ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".webp":
		mimeType = "image/webp"
	case ".svg":
		mimeType = "image/svg+xml"
	case ".gif":
		mimeType = "image/gif"
	}
	base64Str := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Str), nil
}

// AddImageElement adds an image element to the specified slide with default positioning.
func (a *App) AddImageElement(slideIndex int, imageSrc string) engine.Deck {
	if slideIndex < 0 || slideIndex >= len(a.deck.Slides) {
		return a.deck
	}
	el := engine.Element{
		ID:       fmt.Sprintf("el-%d", len(a.deck.Slides[slideIndex].Elements)+1),
		Type:     "image",
		ImageURL: imageSrc,
		Position: engine.Position{X: 200, Y: 120, W: 320, H: 240},
		Style: engine.Style{
			BorderRadius: 8,
			Opacity:      1,
		},
		ZIndex: 5,
	}
	a.deck.Slides[slideIndex].Elements = append(a.deck.Slides[slideIndex].Elements, el)
	a.pushHistory()
	return a.deck
}

// --- AI Operations ---

// AIResult is the return type for the unified AI prompt handler.
type AIResult struct {
	Action  string      `json:"action"`
	Deck    engine.Deck `json:"deck"`
	Message string      `json:"message"`
}

// ProcessAIPrompt is the unified AI handler. It auto-detects the best action
// (generate_deck, add_slide, edit_slide) and applies the result.
func (a *App) ProcessAIPrompt(currentSlideIdx int, prompt string) (AIResult, error) {
	if currentSlideIdx < 0 || currentSlideIdx >= len(a.deck.Slides) {
		currentSlideIdx = 0
	}

	env, err := a.aiAgent.ProcessPromptStream(a.ctx, &a.deck, currentSlideIdx, prompt, func(chunk ai.StreamChunk) {
		runtime.EventsEmit(a.ctx, "ai-stream-chunk", chunk)
	})
	if err != nil {
		return AIResult{Deck: a.deck}, err
	}

	switch env.Action {
	case "generate_deck":
		a.deck = engine.NewDeck()
		if env.Title != "" {
			a.deck.Meta.Title = env.Title
		} else {
			a.deck.Meta.Title = prompt
		}
		if len(env.Slides) > 0 {
			a.deck.Slides = env.Slides
		}
		a.filePath = ""
		a.history = engine.NewHistory(100)
		a.history.Push(a.deck)

	case "add_slide":
		if env.Slide != nil {
			a.deck.Slides = append(a.deck.Slides, *env.Slide)
			a.pushHistory()
		}

	case "edit_slide":
		if env.Slide != nil && currentSlideIdx >= 0 && currentSlideIdx < len(a.deck.Slides) {
			// Preserve the original slide ID
			env.Slide.ID = a.deck.Slides[currentSlideIdx].ID
			a.deck.Slides[currentSlideIdx] = *env.Slide
			a.pushHistory()
		}
	}

	msg := env.Message
	if msg == "" {
		msg = fmt.Sprintf("Completed %s action.", env.Action)
	}

	return AIResult{
		Action:  env.Action,
		Deck:    a.deck,
		Message: msg,
	}, nil
}

// GenerateDeckWithAI creates a new presentation using the AI Agent (legacy wrapper).
func (a *App) GenerateDeckWithAI(prompt string) (engine.Deck, error) {
	res, err := a.ProcessAIPrompt(0, prompt)
	return res.Deck, err
}

// AddSlideWithAI generates a new slide and appends it to the deck using AI (legacy wrapper).
func (a *App) AddSlideWithAI(prompt string) (engine.Deck, error) {
	res, err := a.ProcessAIPrompt(0, prompt)
	return res.Deck, err
}

// --- Internal ---

func (a *App) pushHistory() {
	a.history.Push(a.deck)
}

