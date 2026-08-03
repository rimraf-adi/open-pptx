/**
 * open-pptx — Main Application
 * Hardware-accelerated 60fps canvas editor & AI Co-pilot
 */
import './style.css';

// Wails runtime and Go bindings
import { GetDeck, NewDeck, AddSlide, DeleteSlide, DuplicateSlide, ReorderSlide,
         AddElement, AddShapeElement, UpdateElement, DeleteElement, UpdateDeckMeta, UpdateSlideBg,
         Undo, Redo, CanUndo, CanRedo,
         SaveFile, SaveFileAs, OpenFile, GetFilePath,
         ProcessAIPrompt, GenerateDeckWithAI, AddSlideWithAI } from '../wailsjs/go/main/App';

// ─── State ─────────────────────────────────────────────────────────
let state = {
  deck: null,
  currentSlide: 0,
  selectedElement: null,
  isDragging: false,
  isResizing: false,
  dragOffset: { x: 0, y: 0 },
  resizeHandle: null,
  resizeStart: null,
  canvasScale: 1,
  presenting: false,
  cmdPaletteOpen: false,
  shapeMenuOpen: false,
  fileMenuOpen: false,
  filePath: '',
  contextMenu: null,
  editingText: false,
  // AI Co-pilot state
  aiPanelOpen: false,
  aiLoading: false,
  aiReasoning: '',
  aiStreamingText: '',
  aiContentMode: false, // false = quick prompt, true = content paste
  aiMessages: [
    { role: 'assistant', content: 'Hi! I am your AI Co-pilot. Ask me to generate a presentation deck, add slides, or paste long-form content to auto-generate a full deck.' }
  ],
  // Performance & RAF state
  rafPending: false,
  mousePos: { x: 0, y: 0 },
  activeGuides: { h: null, v: null },
};

let thumbnailTimer = null;

// ─── Initialize ────────────────────────────────────────────────────
async function init() {
  state.deck = await GetDeck();
  renderAppShell();
  setupKeyboardShortcuts();
  window.addEventListener('resize', updateCanvasScale);

  // Subscribe to real-time AI streaming chunks (Reasoning & Content)
  if (window.runtime && window.runtime.EventsOn) {
    window.runtime.EventsOn("ai-stream-chunk", (chunk) => {
      if (chunk) {
        if (chunk.reasoning) {
          state.aiReasoning += chunk.reasoning;
        }
        if (chunk.content) {
          state.aiStreamingText += chunk.content;
        }
        updateAIMessagesOnly();
      }
    });
  }
}

// ─── App Shell Render (Renders base UI container) ──────────────────
function renderAppShell() {
  const app = document.getElementById('app');
  if (!app) return;

  if (state.presenting) {
    app.innerHTML = renderPresentMode();
    setupPresentEvents();
    return;
  }

  app.innerHTML = `
    <div class="titlebar">
      <div class="titlebar-left" style="-webkit-app-region: no-drag;">
        <img class="titlebar-logo" src="/logo.png" alt="open-pptx logo" />
        <span class="titlebar-app-name">open-pptx</span>

        <div class="titlebar-file-wrapper">
          <button class="titlebar-file-btn ${state.fileMenuOpen ? 'active' : ''}" onclick="window.app.toggleFileMenu(event)">
            File ▾
          </button>
          ${state.fileMenuOpen ? `
            <div class="file-menu-popover" onclick="event.stopPropagation()">
              <button class="file-menu-item" onclick="window.app.newDeck()">
                <span>📄 New Presentation</span><span class="file-shortcut">⌘N</span>
              </button>
              <button class="file-menu-item" onclick="window.app.openFile()">
                <span>📂 Open Presentation...</span><span class="file-shortcut">⌘O</span>
              </button>
              <div class="file-menu-sep"></div>
              <button class="file-menu-item" onclick="window.app.saveFile()">
                <span>💾 Save</span><span class="file-shortcut">⌘S</span>
              </button>
              <button class="file-menu-item" onclick="window.app.saveFileAs()">
                <span>💾 Save As...</span><span class="file-shortcut">⌘⇧S</span>
              </button>
            </div>
          ` : ''}
        </div>
      </div>

      <div class="titlebar-center">
        <span class="titlebar-title" id="deckTitleText">${escapeHtml(state.deck?.meta?.title || 'Untitled Presentation')}</span>
        ${state.filePath ? `<span class="titlebar-filepath-badge" title="${escapeHtml(state.filePath)}">● Saved</span>` : `<span class="titlebar-filepath-badge unsaved">● Draft</span>`}
      </div>

      <div class="titlebar-right" style="-webkit-app-region: no-drag;">
        <button class="titlebar-action-btn" onclick="window.app.saveFile()" data-tooltip="Save (⌘S)">
          💾 Save
        </button>
        <button class="titlebar-action-btn" onclick="window.app.openFile()" data-tooltip="Open (⌘O)">
          📂 Open
        </button>
      </div>
    </div>
    <div class="app-layout">
      ${renderSidebar()}
      <div class="canvas-area" id="canvasArea">
        ${renderToolbar()}
        <div class="canvas-wrapper" id="canvasWrapper" style="transform: scale(${state.canvasScale});">
          <div class="slide-canvas" id="slideCanvas" style="background: ${getCurrentSlide().bgColor || '#FFFFFF'};">
            ${renderSlideElements()}
          </div>
        </div>
      </div>
      <div id="propContainer">
        ${state.aiPanelOpen ? renderAIPanel() : renderPropertiesPanel()}
      </div>
    </div>
    <div class="statusbar">
      <span class="statusbar-text" id="statusSlideText">Slide ${state.currentSlide + 1} of ${state.deck.slides.length}</span>
      <div class="statusbar-actions">
        <button class="statusbar-btn" onclick="window.app.present()">▶ Present</button>
        <span class="statusbar-text" id="statusTitleText">${escapeHtml(state.deck.meta.title)}</span>
      </div>
    </div>
    ${!state.aiPanelOpen ? `
      <button class="ai-fab" onclick="window.app.toggleAIPanel()">
        <svg class="ai-sparkle-icon" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2L14.5 9.5L22 12L14.5 14.5L12 22L9.5 14.5L2 12L9.5 9.5L12 2Z"/></svg>
        AI Co-pilot
      </button>
    ` : ''}
    <div id="modalContainer">
      ${state.cmdPaletteOpen ? renderCommandPalette() : ''}
      ${state.contextMenu ? renderContextMenu() : ''}
    </div>
  `;

  setupCanvasEvents();
  updateCanvasScale();
  updateThumbnailsDebounced();
}

// ─── Targeted Canvas & Component Re-renders ───────────────────────
function updateCanvasOnly() {
  const canvas = document.getElementById('slideCanvas');
  if (!canvas) return;
  canvas.style.background = getCurrentSlide().bgColor || '#0F172A';
  canvas.innerHTML = renderSlideElements();

  const statusText = document.getElementById('statusSlideText');
  if (statusText) statusText.innerText = `Slide ${state.currentSlide + 1} of ${state.deck.slides.length}`;
  updatePropertiesOnly();
  updateThumbnailsDebounced();
}

function updatePropertiesOnly() {
  const propContainer = document.getElementById('propContainer');
  if (!propContainer) return;
  propContainer.innerHTML = state.aiPanelOpen ? renderAIPanel() : renderPropertiesPanel();
}

function updateSidebarOnly() {
  const sidebar = document.querySelector('.sidebar');
  if (!sidebar) return;
  const slideList = document.getElementById('slideList');
  if (slideList) {
    slideList.innerHTML = state.deck.slides.map((slide, i) => `
      <div class="slide-thumb ${i === state.currentSlide ? 'active' : ''}"
           data-index="${i}"
           onclick="window.app.selectSlide(${i})"
           oncontextmenu="window.app.showSlideContextMenu(event, ${i})">
        <span class="slide-thumb-number">${i + 1}</span>
        <canvas data-thumb="${i}" width="192" height="108"></canvas>
      </div>
    `).join('');
  }
  updateThumbnailsDebounced();
}

// ─── Sidebar ───────────────────────────────────────────────────────
function renderSidebar() {
  const slides = state.deck.slides;
  return `
    <div class="sidebar">
      <div class="sidebar-header">
        <h3>Slides</h3>
      </div>
      <div class="sidebar-slides" id="slideList">
        ${slides.map((slide, i) => `
          <div class="slide-thumb ${i === state.currentSlide ? 'active' : ''}"
               data-index="${i}"
               onclick="window.app.selectSlide(${i})"
               oncontextmenu="window.app.showSlideContextMenu(event, ${i})">
            <span class="slide-thumb-number">${i + 1}</span>
            <canvas data-thumb="${i}" width="192" height="108"></canvas>
          </div>
        `).join('')}
      </div>
      <button class="sidebar-add-btn" onclick="window.app.addSlide()">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
        Add Slide
      </button>
    </div>
  `;
}

// ─── Preset Color Palette ──────────────────────────────────────────
const presetColors = [
  '#0F172A', '#334155', '#64748B', '#94A3B8', '#F1F5F9', '#FFFFFF',
  '#2563EB', '#3B82F6', '#60A5FA', '#93C5FD', '#DBEAFE',
  '#7C3AED', '#8B5CF6', '#A78BFA', '#C4B5FD', '#EDE9FE',
  '#059669', '#10B981', '#34D399', '#6EE7B7', '#D1FAE5',
  '#D97706', '#F59E0B', '#FBBF24', '#FDE68A', '#FEF3C7',
  '#E11D48', '#F43F5E', '#FB7185', '#FDA4AF', '#FFE4E6',
];

function renderColorSwatches(currentValue, updateFnName) {
  return `
    <div class="color-swatches">
      ${presetColors.map(c => `
        <button class="color-swatch-btn ${c.toLowerCase() === (currentValue || '').toLowerCase() ? 'active' : ''}"
                style="background: ${c};"
                title="${c}"
                onclick="window.app.${updateFnName}('${c}')"></button>
      `).join('')}
    </div>
  `;
}

function renderShapeHTML(shapeType, style) {
  const bg = style.bgColor || '#2563EB';
  const op = style.opacity ?? 1;
  const border = style.borderColor ? `border: ${style.borderWidth || 2}px solid ${style.borderColor};` : '';

  switch (shapeType) {
    case 'circle':
      return `<div class="element-shape" style="background: ${bg}; border-radius: 50%; opacity: ${op}; ${border}"></div>`;
    case 'triangle':
      return `<svg width="100%" height="100%" viewBox="0 0 100 100" preserveAspectRatio="none" style="opacity: ${op};"><polygon points="50,0 100,100 0,100" fill="${bg}"/></svg>`;
    case 'star':
      return `<svg width="100%" height="100%" viewBox="0 0 100 100" preserveAspectRatio="none" style="opacity: ${op};"><polygon points="50,0 63,38 100,38 69,59 82,100 50,75 18,100 31,59 0,38 37,38" fill="${bg}"/></svg>`;
    case 'diamond':
      return `<svg width="100%" height="100%" viewBox="0 0 100 100" preserveAspectRatio="none" style="opacity: ${op};"><polygon points="50,0 100,50 50,100 0,50" fill="${bg}"/></svg>`;
    case 'arrow':
      return `<svg width="100%" height="100%" viewBox="0 0 100 100" preserveAspectRatio="none" style="opacity: ${op};"><polygon points="0,35 60,35 60,10 100,50 60,90 60,65 0,65" fill="${bg}"/></svg>`;
    case 'line':
      return `<div class="element-shape" style="background: ${bg}; border-radius: 4px; opacity: ${op}; height: 100%; width: 100%;"></div>`;
    case 'rect':
    default:
      return `<div class="element-shape" style="background: ${bg}; border-radius: ${style.borderRadius || 8}px; opacity: ${op}; ${border}"></div>`;
  }
}

// ─── Toolbar ───────────────────────────────────────────────────────
function renderToolbar() {
  return `
    <div class="toolbar">
      <button class="toolbar-btn" onclick="window.app.addElement('text')" data-tooltip="Text">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 7V4h16v3M9 20h6M12 4v16"/></svg>
        Text
      </button>

      <div class="toolbar-shape-wrapper">
        <button class="toolbar-btn ${state.shapeMenuOpen ? 'active' : ''}" onclick="window.app.toggleShapeMenu()" data-tooltip="Shapes">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/></svg>
          Shapes ▾
        </button>
        ${state.shapeMenuOpen ? `
          <div class="shape-menu-popover" onclick="event.stopPropagation()">
            <button class="shape-menu-item" onclick="window.app.addShape('rect')">⬛ Rectangle</button>
            <button class="shape-menu-item" onclick="window.app.addShape('circle')">🟡 Circle</button>
            <button class="shape-menu-item" onclick="window.app.addShape('triangle')">🔺 Triangle</button>
            <button class="shape-menu-item" onclick="window.app.addShape('star')">⭐ Star</button>
            <button class="shape-menu-item" onclick="window.app.addShape('diamond')">◆ Diamond</button>
            <button class="shape-menu-item" onclick="window.app.addShape('arrow')">➔ Arrow</button>
            <button class="shape-menu-item" onclick="window.app.addShape('line')">➖ Line</button>
          </div>
        ` : ''}
      </div>

      <button class="toolbar-btn" onclick="window.app.addElement('code')" data-tooltip="Code Block">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>
        Code
      </button>
      <div class="toolbar-sep"></div>
      <button class="toolbar-btn" onclick="window.app.undo()" data-tooltip="Undo (⌘Z)">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/></svg>
      </button>
      <button class="toolbar-btn" onclick="window.app.redo()" data-tooltip="Redo (⌘⇧Z)">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.13-9.36L23 10"/></svg>
      </button>
      <div class="toolbar-sep"></div>
      <button class="toolbar-btn ${state.aiPanelOpen ? 'active' : ''}" onclick="window.app.toggleAIPanel()" data-tooltip="AI Co-pilot">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2L14.5 9.5L22 12L14.5 14.5L12 22L9.5 14.5L2 12L9.5 9.5L12 2Z"/></svg>
        Ask AI
      </button>
      <div class="toolbar-sep"></div>
      <button class="toolbar-btn" onclick="window.app.present()" data-tooltip="Present (⌘⏎)">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>
      </button>
    </div>
  `;
}

// ─── Slide Elements ────────────────────────────────────────────────
function renderSlideElements() {
  const slide = getCurrentSlide();
  if (!slide || !slide.elements) return '';

  let html = slide.elements.map(el => {
    const isSelected = state.selectedElement === el.id;
    const isEditing = state.editingText && isSelected;
    const pos = el.position;
    const style = el.style || {};

    let content = '';
    switch (el.type) {
      case 'text':
        content = `<div class="element-text ${isEditing ? 'editing' : ''}" 
          ${isEditing ? 'contenteditable="true"' : ''}
          style="font-size: ${style.fontSize || 24}px; font-weight: ${style.fontWeight || 'normal'}; color: ${style.color || '#0F172A'}; text-align: ${style.textAlign || 'left'}; font-family: ${style.fontFamily || 'Inter'}, sans-serif; line-height: ${style.lineHeight || 1.4};"
          ondblclick="window.app.startTextEdit('${el.id}')"
          onblur="window.app.finishTextEdit('${el.id}', this.innerText)"
        >${escapeHtml(el.content || 'Click to edit text')}</div>`;
        break;
      case 'shape':
        content = renderShapeHTML(el.shapeType, style);
        break;
      case 'code':
        content = `<div class="element-code" style="font-size: ${style.fontSize || 14}px; color: ${style.color || '#0F172A'}; background: ${style.bgColor || '#F1F5F9'}; border-radius: ${style.borderRadius || 12}px; font-family: ${style.fontFamily || "'JetBrains Mono', monospace"};">${escapeHtml(el.content || '// your code here')}</div>`;
        break;
      case 'image':
        content = el.imageUrl
          ? `<img src="${el.imageUrl}" style="width: 100%; height: 100%; object-fit: cover; border-radius: ${style.borderRadius || 0}px;" />`
          : `<div class="empty-state" style="height: 100%;"><span style="font-size: 24px;">🖼</span><span style="font-size: 11px;">No image</span></div>`;
        break;
    }

    return `
      <div class="slide-element ${isSelected ? 'selected' : ''}"
           data-element-id="${el.id}"
           style="left: ${pos.x}px; top: ${pos.y}px; width: ${pos.w}px; height: ${pos.h}px; z-index: ${el.zIndex || 1};"
           onmousedown="window.app.startDrag(event, '${el.id}')"
           onclick="window.app.selectElement(event, '${el.id}')"
           oncontextmenu="window.app.showElementContextMenu(event, '${el.id}')">
        ${content}
        ${isSelected ? `
          <div class="resize-handle nw" onmousedown="window.app.startResize(event, '${el.id}', 'nw')"></div>
          <div class="resize-handle ne" onmousedown="window.app.startResize(event, '${el.id}', 'ne')"></div>
          <div class="resize-handle sw" onmousedown="window.app.startResize(event, '${el.id}', 'sw')"></div>
          <div class="resize-handle se" onmousedown="window.app.startResize(event, '${el.id}', 'se')"></div>
        ` : ''}
      </div>
    `;
  }).join('');

  // Guide lines
  if (state.activeGuides.h !== null) {
    html += `<div class="align-guide-h" style="top: ${state.activeGuides.h}px;"></div>`;
  }
  if (state.activeGuides.v !== null) {
    html += `<div class="align-guide-v" style="left: ${state.activeGuides.v}px;"></div>`;
  }

  return html;
}

// ─── Properties Panel ──────────────────────────────────────────────
function renderPropertiesPanel() {
  const el = getSelectedElement();
  if (!el) {
    return `
      <div class="properties-panel">
        <div class="properties-header"><h3>Properties</h3></div>
        <div class="prop-section">
          <div class="prop-section-title">Slide Background</div>
          <div class="prop-row" style="flex-direction: column; align-items: stretch; gap: 4px;">
            <div style="display: flex; align-items: center; justify-content: space-between;">
              <span class="prop-label">Color</span>
              <input type="color" class="prop-color-input" value="${getCurrentSlide().bgColor || '#FFFFFF'}" onchange="window.app.updateSlideBg(this.value)" />
            </div>
            ${renderColorSwatches(getCurrentSlide().bgColor || '#FFFFFF', 'updateSlideBg')}
          </div>
        </div>
        <div class="prop-section">
          <div class="prop-section-title">Deck</div>
          <div class="prop-row">
            <span class="prop-label">Title</span>
            <input type="text" class="prop-input" value="${escapeHtml(state.deck.meta.title)}" onchange="window.app.updateTitle(this.value)" />
          </div>
        </div>
      </div>
    `;
  }

  const pos = el.position;
  const style = el.style || {};

  let typeSpecific = '';
  if (el.type === 'text') {
    typeSpecific = `
      <div class="prop-section">
        <div class="prop-section-title">Typography</div>

        <div class="prop-row">
          <span class="prop-label">Font</span>
          <select class="prop-select" onchange="window.app.updateStyle('fontFamily', this.value)">
            <option value="Inter" ${(style.fontFamily || 'Inter') === 'Inter' ? 'selected' : ''}>Inter (Sans)</option>
            <option value="Roboto" ${style.fontFamily === 'Roboto' ? 'selected' : ''}>Roboto</option>
            <option value="JetBrains Mono" ${style.fontFamily === 'JetBrains Mono' ? 'selected' : ''}>JetBrains Mono</option>
            <option value="Outfit" ${style.fontFamily === 'Outfit' ? 'selected' : ''}>Outfit</option>
            <option value="Playfair Display" ${style.fontFamily === 'Playfair Display' ? 'selected' : ''}>Playfair (Serif)</option>
          </select>
        </div>

        <div class="prop-row">
          <span class="prop-label">Size</span>
          <div class="format-btn-group">
            <button class="format-btn" onclick="window.app.stepFontSize(-2)">-</button>
            <input type="number" id="propFontSize" class="prop-input" style="text-align: center; width: 50px;" value="${style.fontSize || 24}" onchange="window.app.updateStyle('fontSize', parseInt(this.value))" />
            <button class="format-btn" onclick="window.app.stepFontSize(2)">+</button>
          </div>
        </div>

        <div class="prop-row">
          <span class="prop-label">Weight</span>
          <div class="format-btn-group">
            <button class="format-btn ${(style.fontWeight || 'normal') === 'normal' ? 'active' : ''}" onclick="window.app.updateStyle('fontWeight', 'normal')">Regular</button>
            <button class="format-btn ${style.fontWeight === '600' ? 'active' : ''}" onclick="window.app.updateStyle('fontWeight', '600')">Semi</button>
            <button class="format-btn ${style.fontWeight === 'bold' ? 'active' : ''}" onclick="window.app.updateStyle('fontWeight', 'bold')">Bold</button>
          </div>
        </div>

        <div class="prop-row">
          <span class="prop-label">Align</span>
          <div class="format-btn-group">
            <button class="format-btn ${(style.textAlign || 'left') === 'left' ? 'active' : ''}" onclick="window.app.updateStyle('textAlign', 'left')">⬅ Left</button>
            <button class="format-btn ${style.textAlign === 'center' ? 'active' : ''}" onclick="window.app.updateStyle('textAlign', 'center')">⏺ Center</button>
            <button class="format-btn ${style.textAlign === 'right' ? 'active' : ''}" onclick="window.app.updateStyle('textAlign', 'right')">Right ➡️</button>
          </div>
        </div>

        <div class="prop-row" style="flex-direction: column; align-items: stretch; gap: 4px; margin-top: 8px;">
          <div style="display: flex; align-items: center; justify-content: space-between;">
            <span class="prop-label">Text Color</span>
            <input type="color" class="prop-color-input" value="${style.color || '#0F172A'}" onchange="window.app.updateStyle('color', this.value)" />
          </div>
          ${renderColorSwatches(style.color || '#0F172A', 'updateTextColor')}
        </div>
      </div>
    `;
  } else if (el.type === 'shape') {
    typeSpecific = `
      <div class="prop-section">
        <div class="prop-section-title">Shape Type</div>
        <div class="shape-picker-grid">
          <button class="shape-picker-item ${(el.shapeType || 'rect') === 'rect' ? 'active' : ''}" title="Rectangle" onclick="window.app.updateShapeType('rect')">⬛</button>
          <button class="shape-picker-item ${el.shapeType === 'circle' ? 'active' : ''}" title="Circle" onclick="window.app.updateShapeType('circle')">🟡</button>
          <button class="shape-picker-item ${el.shapeType === 'triangle' ? 'active' : ''}" title="Triangle" onclick="window.app.updateShapeType('triangle')">🔺</button>
          <button class="shape-picker-item ${el.shapeType === 'star' ? 'active' : ''}" title="Star" onclick="window.app.updateShapeType('star')">⭐</button>
          <button class="shape-picker-item ${el.shapeType === 'diamond' ? 'active' : ''}" title="Diamond" onclick="window.app.updateShapeType('diamond')">◆</button>
          <button class="shape-picker-item ${el.shapeType === 'arrow' ? 'active' : ''}" title="Arrow" onclick="window.app.updateShapeType('arrow')">➔</button>
          <button class="shape-picker-item ${el.shapeType === 'line' ? 'active' : ''}" title="Line" onclick="window.app.updateShapeType('line')">➖</button>
        </div>
      </div>

      <div class="prop-section">
        <div class="prop-section-title">Fill & Style</div>
        <div class="prop-row" style="flex-direction: column; align-items: stretch; gap: 4px;">
          <div style="display: flex; align-items: center; justify-content: space-between;">
            <span class="prop-label">Fill Color</span>
            <input type="color" class="prop-color-input" value="${style.bgColor || '#2563EB'}" onchange="window.app.updateStyle('bgColor', this.value)" />
          </div>
          ${renderColorSwatches(style.bgColor || '#2563EB', 'updateShapeColor')}
        </div>

        <div class="prop-row" style="margin-top: 8px;">
          <span class="prop-label">Radius</span>
          <input type="range" class="prop-slider" min="0" max="40" value="${style.borderRadius || 8}" oninput="window.app.updateStyle('borderRadius', parseInt(this.value))" />
        </div>
        <div class="prop-row">
          <span class="prop-label">Opacity</span>
          <input type="range" class="prop-slider" min="0" max="100" value="${(style.opacity ?? 1) * 100}" oninput="window.app.updateStyle('opacity', parseInt(this.value) / 100)" />
        </div>
      </div>
    `;
  }

  return `
    <div class="properties-panel">
      <div class="properties-header"><h3>Properties</h3></div>
      <div class="prop-section">
        <div class="prop-section-title">Position</div>
        <div class="prop-row">
          <span class="prop-label">X</span>
          <input type="number" id="propPosX" class="prop-input" value="${Math.round(pos.x)}" onchange="window.app.updatePosition('x', parseFloat(this.value))" />
          <span class="prop-label">Y</span>
          <input type="number" id="propPosY" class="prop-input" value="${Math.round(pos.y)}" onchange="window.app.updatePosition('y', parseFloat(this.value))" />
        </div>
        <div class="prop-row">
          <span class="prop-label">W</span>
          <input type="number" id="propPosW" class="prop-input" value="${Math.round(pos.w)}" onchange="window.app.updatePosition('w', parseFloat(this.value))" />
          <span class="prop-label">H</span>
          <input type="number" id="propPosH" class="prop-input" value="${Math.round(pos.h)}" onchange="window.app.updatePosition('h', parseFloat(this.value))" />
        </div>
      </div>
      ${typeSpecific}
    </div>
  `;
}

function updateAIMessagesOnly() {
  const msgList = document.getElementById('aiMsgList');
  if (!msgList) return;
  msgList.innerHTML = renderAIMessageItems();
  msgList.scrollTop = msgList.scrollHeight;
}

function renderAIMessageItems() {
  let html = state.aiMessages.map((msg, i) => `
    <div class="ai-msg ${msg.role}">
      ${msg.reasoning ? `
        <div class="ai-reasoning-box">
          <div class="ai-reasoning-header" onclick="window.app.toggleMsgReasoning(${i})">
            <span class="ai-reasoning-title">🧠 Model Reasoning</span>
            <span>${msg.showReasoning !== false ? '▲' : '▼'}</span>
          </div>
          ${msg.showReasoning !== false ? `<div class="ai-reasoning-body">${escapeHtml(msg.reasoning)}</div>` : ''}
        </div>
      ` : ''}
      <div class="ai-msg-bubble">${escapeHtml(msg.content)}</div>
      <div class="ai-msg-actions">
        <button class="ai-copy-btn" onclick="window.app.copyMsg(${i}, event)">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
          Copy
        </button>
      </div>
    </div>
  `).join('');

  if (state.aiLoading) {
    html += `
      <div class="ai-msg assistant">
        ${state.aiReasoning ? `
          <div class="ai-reasoning-box">
            <div class="ai-reasoning-header">
              <span class="ai-reasoning-title">
                <span class="ai-reasoning-pulse"></span>
                🧠 Model Reasoning (Streaming...)
              </span>
            </div>
            <div class="ai-reasoning-body">${escapeHtml(state.aiReasoning)}</div>
          </div>
        ` : ''}
        <div class="ai-msg-bubble">
          ${state.aiStreamingText ? escapeHtml(state.aiStreamingText) : `
            <div class="ai-shimmer">
              <div class="ai-shimmer-dot"></div>
              <div class="ai-shimmer-dot"></div>
              <div class="ai-shimmer-dot"></div>
            </div>
          `}
        </div>
      </div>
    `;
  }

  return html;
}

// ─── AI Co-Pilot Panel ─────────────────────────────────────────────
function renderAIPanel() {
  return `
    <div class="ai-panel">
      <div class="ai-panel-header">
        <div class="ai-panel-title">
          <svg class="ai-sparkle-icon" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2L14.5 9.5L22 12L14.5 14.5L12 22L9.5 14.5L2 12L9.5 9.5L12 2Z"/></svg>
          AI Co-pilot
        </div>
        <button class="ai-close-btn" onclick="window.app.toggleAIPanel()">✕</button>
      </div>

      <!-- Mode Tabs -->
      <div class="ai-mode-tabs">
        <button class="ai-mode-tab ${!state.aiContentMode ? 'active' : ''}" onclick="window.app.setAIMode(false)">
          ⚡ Quick Prompt
        </button>
        <button class="ai-mode-tab ${state.aiContentMode ? 'active' : ''}" onclick="window.app.setAIMode(true)">
          📄 Content → Deck
        </button>
      </div>

      ${state.aiContentMode ? renderContentMode() : renderQuickPromptMode()}
    </div>
  `;
}

function renderQuickPromptMode() {
  return `
    <div class="ai-messages" id="aiMsgList">
      ${renderAIMessageItems()}
    </div>

    <div class="ai-pills">
      <button class="ai-pill" onclick="window.app.sendAIPrompt('Create a 5-slide pitch deck for a startup')">🚀 Pitch Deck</button>
      <button class="ai-pill" onclick="window.app.sendAIPrompt('Add a slide comparing our pricing tiers')">💰 Pricing</button>
      <button class="ai-pill" onclick="window.app.sendAIPrompt('Add a slide showing high level architecture')">🏗️ Architecture</button>
      <button class="ai-pill" onclick="window.app.sendAIPrompt('@slide${state.currentSlide + 1} make this slide more visually appealing')">✨ Beautify Current</button>
    </div>

    <div class="ai-input-box">
      <input class="ai-input" id="aiInput" type="text" placeholder="Ask AI... use @slide3 to target a slide" onkeydown="if(event.key==='Enter') window.app.submitAIPrompt()" />
      <button class="ai-send-btn" onclick="window.app.submitAIPrompt()" ${state.aiLoading ? 'disabled' : ''}>Send</button>
    </div>
  `;
}

function renderContentMode() {
  return `
    <div class="ai-content-section">
      <div class="ai-content-header">
        <span class="ai-content-header-icon">📋</span>
        <div>
          <div class="ai-content-title">Paste your content</div>
          <div class="ai-content-subtitle">Notes, outlines, docs, or any text — AI will analyze and build a complete presentation.</div>
        </div>
      </div>

      <textarea
        class="ai-content-textarea"
        id="aiContentTextarea"
        placeholder="Paste your content here...

Examples:
• Meeting notes or transcripts
• Product specs or PRDs
• Blog posts or articles
• Research summaries
• Bullet-point outlines
• Any raw text content

The AI will analyze the structure, extract key points, and generate a visually stunning multi-slide deck."
        rows="12"
      >${state.aiContentText || ''}</textarea>

      <div class="ai-content-options">
        <div class="ai-content-option-row">
          <span class="prop-label">Slides</span>
          <select class="prop-select" id="aiContentSlideCount">
            <option value="auto">Auto (AI decides)</option>
            <option value="3">3 slides</option>
            <option value="5" selected>5 slides</option>
            <option value="8">8 slides</option>
            <option value="10">10 slides</option>
          </select>
        </div>
        <div class="ai-content-option-row">
          <span class="prop-label">Style</span>
          <select class="prop-select" id="aiContentStyle">
            <option value="modern" selected>Modern & Clean</option>
            <option value="bold">Bold & Impactful</option>
            <option value="minimal">Minimal</option>
            <option value="corporate">Corporate</option>
          </select>
        </div>
      </div>

      <button
        class="ai-content-generate-btn"
        onclick="window.app.generateFromContent()"
        ${state.aiLoading ? 'disabled' : ''}
      >
        ${state.aiLoading
          ? '<span class="ai-shimmer"><span class="ai-shimmer-dot"></span><span class="ai-shimmer-dot"></span><span class="ai-shimmer-dot"></span></span> Generating...'
          : '✨ Generate Presentation'}
      </button>

      ${state.aiLoading && state.aiReasoning ? `
        <div class="ai-reasoning-live">
          <div class="ai-reasoning-label">🧠 AI is thinking...</div>
          <div class="ai-reasoning-text">${escapeHtml(state.aiReasoning.slice(-400))}</div>
        </div>
      ` : ''}
    </div>
  `;
}

// ─── Command Palette ───────────────────────────────────────────────
const commands = [
  { name: 'New Presentation', shortcut: '⌘N', action: () => window.app.newDeck() },
  { name: 'Open File', shortcut: '⌘O', action: () => window.app.openFile() },
  { name: 'Save', shortcut: '⌘S', action: () => window.app.saveFile() },
  { name: 'Save As...', shortcut: '⌘⇧S', action: () => window.app.saveFileAs() },
  { name: 'Add Slide', shortcut: '', action: () => window.app.addSlide() },
  { name: 'Add Text', shortcut: 'T', action: () => window.app.addElement('text') },
  { name: 'Add Shape', shortcut: 'S', action: () => window.app.addElement('shape') },
  { name: 'Add Code Block', shortcut: '', action: () => window.app.addElement('code') },
  { name: 'Ask AI Co-pilot', shortcut: '⌘A', action: () => window.app.toggleAIPanel() },
  { name: 'Present', shortcut: '⌘⏎', action: () => window.app.present() },
  { name: 'Undo', shortcut: '⌘Z', action: () => window.app.undo() },
  { name: 'Redo', shortcut: '⌘⇧Z', action: () => window.app.redo() },
  { name: 'Delete Selected', shortcut: '⌫', action: () => window.app.deleteSelected() },
  { name: 'Duplicate Slide', shortcut: '⌘D', action: () => window.app.duplicateSlide() },
];

function renderCommandPalette() {
  return `
    <div class="cmd-palette-overlay" onclick="window.app.closeCmdPalette()">
      <div class="cmd-palette" onclick="event.stopPropagation()">
        <input class="cmd-palette-input" id="cmdInput" type="text" placeholder="Type a command..." oninput="window.app.filterCommands(this.value)" autofocus />
        <div class="cmd-palette-results" id="cmdResults">
          ${commands.map((cmd, i) => `
            <div class="cmd-palette-item ${i === 0 ? 'highlighted' : ''}" onclick="window.app.runCommand(${commands.indexOf(cmd)})">
              <span class="cmd-palette-item-text">${cmd.name}</span>
              ${cmd.shortcut ? `<span class="cmd-palette-item-shortcut">${cmd.shortcut}</span>` : ''}
            </div>
          `).join('')}
        </div>
      </div>
    </div>
  `;
}

// ─── Context Menu ──────────────────────────────────────────────────
function renderContextMenu() {
  const cm = state.contextMenu;
  return `
    <div class="context-menu" style="left: ${cm.x}px; top: ${cm.y}px;">
      ${cm.items.map((item, i) => {
        if (item.sep) return '<div class="context-menu-sep"></div>';
        return `<div class="context-menu-item ${item.danger ? 'danger' : ''}" onclick="window.app.runContextAction(${i})">${item.label}</div>`;
      }).join('')}
    </div>
  `;
}

// ─── Present Mode ──────────────────────────────────────────────────
function renderPresentMode() {
  const slide = getCurrentSlide();
  const scaleX = window.innerWidth / 960;
  const scaleY = window.innerHeight / 540;
  const scale = Math.min(scaleX, scaleY);

  return `
    <div class="present-overlay" id="presentOverlay">
      <div class="present-slide" style="width: 960px; height: 540px; transform: scale(${scale}); transform-origin: center center; background: ${slide.bgColor || '#FFFFFF'};">
        ${(slide.elements || []).map(el => {
          const pos = el.position;
          const style = el.style || {};
          let content = '';
          switch (el.type) {
            case 'text':
              content = `<div style="font-size: ${style.fontSize || 24}px; font-weight: ${style.fontWeight || 'normal'}; color: ${style.color || '#0F172A'}; text-align: ${style.textAlign || 'left'}; font-family: ${style.fontFamily || 'Inter'}, sans-serif; line-height: ${style.lineHeight || 1.4}; width: 100%; height: 100%;">${el.content}</div>`;
              break;
            case 'shape':
              content = renderShapeHTML(el.shapeType, style);
              break;
            case 'code':
              content = `<div style="width: 100%; height: 100%; padding: 16px; font-size: ${style.fontSize || 14}px; color: ${style.color || '#E2E8F0'}; background: ${style.bgColor || '#1E293B'}; border-radius: ${style.borderRadius || 12}px; font-family: 'JetBrains Mono', monospace; white-space: pre; overflow: auto; line-height: 1.6;">${escapeHtml(el.content)}</div>`;
              break;
          }
          return `<div style="position: absolute; left: ${pos.x}px; top: ${pos.y}px; width: ${pos.w}px; height: ${pos.h}px; z-index: ${el.zIndex || 1};">${content}</div>`;
        }).join('')}
      </div>
      <div class="present-hud">
        <button class="present-hud-btn" onclick="window.app.presentPrev()">‹</button>
        <span class="present-hud-text">${state.currentSlide + 1} / ${state.deck.slides.length}</span>
        <button class="present-hud-btn" onclick="window.app.presentNext()">›</button>
        <button class="present-hud-btn" onclick="window.app.exitPresent()">✕</button>
      </div>
    </div>
  `;
}

// ─── Event Setup ───────────────────────────────────────────────────
function setupCanvasEvents() {
  const canvas = document.getElementById('slideCanvas');
  if (!canvas) return;

  canvas.addEventListener('click', (e) => {
    if (e.target === canvas || e.target.classList.contains('slide-canvas')) {
      state.selectedElement = null;
      state.contextMenu = null;
      updateCanvasOnly();
    }
  });

  document.addEventListener('click', () => {
    if (state.contextMenu) {
      state.contextMenu = null;
      const modal = document.getElementById('modalContainer');
      if (modal) modal.innerHTML = '';
    }
  });

  document.addEventListener('mousemove', onMouseMove);
  document.addEventListener('mouseup', onMouseUp);
}

function setupPresentEvents() {
  document.addEventListener('keydown', function presentKey(e) {
    if (!state.presenting) {
      document.removeEventListener('keydown', presentKey);
      return;
    }
    if (e.key === 'Escape') { window.app.exitPresent(); }
    else if (e.key === 'ArrowRight' || e.key === ' ') { window.app.presentNext(); }
    else if (e.key === 'ArrowLeft') { window.app.presentPrev(); }
  });
}

function setupKeyboardShortcuts() {
  document.addEventListener('keydown', (e) => {
    if (state.presenting) return;
    if (state.editingText) return;

    const isMod = e.metaKey || e.ctrlKey;

    if (isMod && e.key === 'k') {
      e.preventDefault();
      window.app.toggleCmdPalette();
      return;
    }

    if (isMod && e.key === 's') {
      e.preventDefault();
      if (e.shiftKey) window.app.saveFileAs();
      else window.app.saveFile();
      return;
    }

    if (isMod && e.key === 'o') {
      e.preventDefault();
      window.app.openFile();
      return;
    }

    if (isMod && e.key === 'n') {
      e.preventDefault();
      window.app.newDeck();
      return;
    }

    if (isMod && e.key === 'z') {
      e.preventDefault();
      if (e.shiftKey) window.app.redo();
      else window.app.undo();
      return;
    }

    if ((e.key === 'Backspace' || e.key === 'Delete') && state.selectedElement) {
      e.preventDefault();
      window.app.deleteSelected();
      return;
    }

    if (isMod && e.key === 'Enter') {
      e.preventDefault();
      window.app.present();
      return;
    }

    if (isMod && e.key === 'd') {
      e.preventDefault();
      window.app.duplicateSlide();
      return;
    }

    if (e.key === 'Escape') {
      if (state.cmdPaletteOpen) {
        window.app.closeCmdPalette();
      } else if (state.selectedElement) {
        state.selectedElement = null;
        updateCanvasOnly();
      }
      return;
    }

    if (state.selectedElement && ['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'].includes(e.key)) {
      e.preventDefault();
      const step = e.shiftKey ? 10 : 1;
      const el = getSelectedElement();
      if (!el) return;
      const newPos = { ...el.position };
      if (e.key === 'ArrowUp') newPos.y -= step;
      if (e.key === 'ArrowDown') newPos.y += step;
      if (e.key === 'ArrowLeft') newPos.x -= step;
      if (e.key === 'ArrowRight') newPos.x += step;
      el.position = newPos;
      UpdateElement(state.currentSlide, el).then(deck => {
        state.deck = deck;
        updateCanvasOnly();
      });
    }
  });
}

// ─── Canvas Scaling ────────────────────────────────────────────────
function updateCanvasScale() {
  const area = document.getElementById('canvasArea');
  const wrapper = document.getElementById('canvasWrapper');
  if (!area || !wrapper) return;

  const padding = 80;
  const availW = area.clientWidth - padding;
  const availH = area.clientHeight - padding;
  const scaleX = availW / 960;
  const scaleY = availH / 540;
  state.canvasScale = Math.min(scaleX, scaleY, 1);

  wrapper.style.transform = `scale(${state.canvasScale})`;
}

// ─── Thumbnails (Debounced) ────────────────────────────────────────
function updateThumbnailsDebounced() {
  if (thumbnailTimer) clearTimeout(thumbnailTimer);
  thumbnailTimer = setTimeout(() => {
    requestAnimationFrame(updateThumbnails);
  }, 100);
}

function updateThumbnails() {
  state.deck.slides.forEach((slide, i) => {
    const canvas = document.querySelector(`canvas[data-thumb="${i}"]`);
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    const scale = 192 / 960;

    ctx.fillStyle = slide.bgColor || '#FFFFFF';
    ctx.fillRect(0, 0, 192, 108);

    (slide.elements || []).forEach(el => {
      const x = el.position.x * scale;
      const y = el.position.y * scale;
      const w = el.position.w * scale;
      const h = el.position.h * scale;

      if (el.type === 'text') {
        ctx.fillStyle = el.style?.color || '#0F172A';
        const fontSize = Math.max(4, (el.style?.fontSize || 24) * scale);
        ctx.font = `${el.style?.fontWeight || 'normal'} ${fontSize}px Inter, sans-serif`;
        ctx.textAlign = el.style?.textAlign || 'left';
        const textX = el.style?.textAlign === 'center' ? x + w / 2 : el.style?.textAlign === 'right' ? x + w : x;
        ctx.fillText(el.content, textX, y + fontSize);
      } else if (el.type === 'shape') {
        ctx.fillStyle = el.style?.bgColor || '#2563EB';
        ctx.globalAlpha = el.style?.opacity ?? 1;
        const r = Math.min((el.style?.borderRadius || 8) * scale, w / 2, h / 2);
        roundRect(ctx, x, y, w, h, r);
        ctx.fill();
        ctx.globalAlpha = 1;
      } else if (el.type === 'code') {
        ctx.fillStyle = el.style?.bgColor || '#F1F5F9';
        const r = Math.min((el.style?.borderRadius || 12) * scale, w / 2, h / 2);
        roundRect(ctx, x, y, w, h, r);
        ctx.fill();
      }
    });
  });
}

function roundRect(ctx, x, y, w, h, r) {
  ctx.beginPath();
  ctx.moveTo(x + r, y);
  ctx.lineTo(x + w - r, y);
  ctx.quadraticCurveTo(x + w, y, x + w, y + r);
  ctx.lineTo(x + w, y + h - r);
  ctx.quadraticCurveTo(x + w, y + h, x + w - r, y + h);
  ctx.lineTo(x + r, y + h);
  ctx.quadraticCurveTo(x, y + h, x, y + h - r);
  ctx.lineTo(x, y + r);
  ctx.quadraticCurveTo(x, y, x + r, y);
  ctx.closePath();
}

// ─── Hardware-Accelerated 60 FPS Drag & Resize ──────────────────────
function onMouseMove(e) {
  if (!state.isDragging && !state.isResizing) return;

  state.mousePos = { x: e.clientX, y: e.clientY };

  if (!state.rafPending) {
    state.rafPending = true;
    requestAnimationFrame(processDragFrame);
  }
}

function processDragFrame() {
  state.rafPending = false;
  const el = getSelectedElement();
  if (!el) return;

  const scale = state.canvasScale;
  const canvas = document.getElementById('slideCanvas');
  const rect = canvas ? canvas.getBoundingClientRect() : { left: 0, top: 0 };

  state.activeGuides = { h: null, v: null };

  if (state.isDragging) {
    const currentCanvasX = (state.mousePos.x - rect.left) / scale;
    const currentCanvasY = (state.mousePos.y - rect.top) / scale;

    let newX = Math.round(currentCanvasX - state.dragOffset.x);
    let newY = Math.round(currentCanvasY - state.dragOffset.y);

    // Smart Snapping Alignment Guidelines
    const snapThreshold = 6;
    const centerX = newX + el.position.w / 2;
    const centerY = newY + el.position.h / 2;

    // Snap to slide center horizontal (Y=270)
    if (Math.abs(centerY - 270) < snapThreshold) {
      newY = Math.round(270 - el.position.h / 2);
      state.activeGuides.h = 270;
    }
    // Snap to slide center vertical (X=480)
    if (Math.abs(centerX - 480) < snapThreshold) {
      newX = Math.round(480 - el.position.w / 2);
      state.activeGuides.v = 480;
    }

    el.position.x = newX;
    el.position.y = newY;

    const dom = document.querySelector(`[data-element-id="${el.id}"]`);
    if (dom) {
      dom.style.left = `${newX}px`;
      dom.style.top = `${newY}px`;
    }

    // Update Property inputs live without re-rendering HTML
    const posXInput = document.getElementById('propPosX');
    const posYInput = document.getElementById('propPosY');
    if (posXInput) posXInput.value = newX;
    if (posYInput) posYInput.value = newY;
  }

  if (state.isResizing) {
    const dx = (state.mousePos.x - state.resizeStart.x) / scale;
    const dy = (state.mousePos.y - state.resizeStart.y) / scale;
    const origPos = state.resizeStart.origPos;

    let newPos = { ...origPos };
    switch (state.resizeHandle) {
      case 'se':
        newPos.w = Math.max(30, Math.round(origPos.w + dx));
        newPos.h = Math.max(30, Math.round(origPos.h + dy));
        break;
      case 'sw':
        newPos.x = Math.round(origPos.x + dx);
        newPos.w = Math.max(30, Math.round(origPos.w - dx));
        newPos.h = Math.max(30, Math.round(origPos.h + dy));
        break;
      case 'ne':
        newPos.y = Math.round(origPos.y + dy);
        newPos.w = Math.max(30, Math.round(origPos.w + dx));
        newPos.h = Math.max(30, Math.round(origPos.h - dy));
        break;
      case 'nw':
        newPos.x = Math.round(origPos.x + dx);
        newPos.y = Math.round(origPos.y + dy);
        newPos.w = Math.max(30, Math.round(origPos.w - dx));
        newPos.h = Math.max(30, Math.round(origPos.h - dy));
        break;
    }
    el.position = newPos;

    const dom = document.querySelector(`[data-element-id="${el.id}"]`);
    if (dom) {
      dom.style.left = `${newPos.x}px`;
      dom.style.top = `${newPos.y}px`;
      dom.style.width = `${newPos.w}px`;
      dom.style.height = `${newPos.h}px`;
    }

    const posXInput = document.getElementById('propPosX');
    const posYInput = document.getElementById('propPosY');
    const posWInput = document.getElementById('propPosW');
    const posHInput = document.getElementById('propPosH');
    if (posXInput) posXInput.value = newPos.x;
    if (posYInput) posYInput.value = newPos.y;
    if (posWInput) posWInput.value = newPos.w;
    if (posHInput) posHInput.value = newPos.h;
  }

  // Update guide lines
  let guideH = document.querySelector('.align-guide-h');
  if (state.activeGuides.h !== null) {
    if (!guideH) {
      guideH = document.createElement('div');
      guideH.className = 'align-guide-h';
      canvas.appendChild(guideH);
    }
    guideH.style.top = `${state.activeGuides.h}px`;
  } else if (guideH) {
    guideH.remove();
  }

  let guideV = document.querySelector('.align-guide-v');
  if (state.activeGuides.v !== null) {
    if (!guideV) {
      guideV = document.createElement('div');
      guideV.className = 'align-guide-v';
      canvas.appendChild(guideV);
    }
    guideV.style.left = `${state.activeGuides.v}px`;
  } else if (guideV) {
    guideV.remove();
  }
}

async function onMouseUp() {
  if (state.isDragging || state.isResizing) {
    state.isDragging = false;
    state.isResizing = false;
    state.activeGuides = { h: null, v: null };

    // Remove guide elements
    document.querySelectorAll('.align-guide-h, .align-guide-v').forEach(el => el.remove());

    const el = getSelectedElement();
    if (el) {
      state.deck = await UpdateElement(state.currentSlide, el);
    }
    updateThumbnailsDebounced();
  }
}

// ─── Helpers ───────────────────────────────────────────────────────
function getCurrentSlide() {
  return state.deck.slides[state.currentSlide] || state.deck.slides[0];
}

function getSelectedElement() {
  if (!state.selectedElement) return null;
  const slide = getCurrentSlide();
  return slide.elements?.find(el => el.id === state.selectedElement) || null;
}

function escapeHtml(str) {
  if (!str) return '';
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

// ─── Public API (window.app) ───────────────────────────────────────
window.app = {
  // AI Co-pilot actions
  toggleAIPanel() {
    state.aiPanelOpen = !state.aiPanelOpen;
    renderAppShell();
  },

  setAIMode(isContentMode) {
    state.aiContentMode = isContentMode;
    updatePropertiesOnly();
  },

  async generateFromContent() {
    const textarea = document.getElementById('aiContentTextarea');
    const slideCountSelect = document.getElementById('aiContentSlideCount');
    const styleSelect = document.getElementById('aiContentStyle');

    const content = textarea?.value?.trim();
    if (!content || state.aiLoading) return;

    const slideCount = slideCountSelect?.value || 'auto';
    const designStyle = styleSelect?.value || 'modern';

    state.aiContentText = content;
    state.aiLoading = true;
    state.aiReasoning = '';
    state.aiStreamingText = '';
    updatePropertiesOnly();

    const prompt = `CONTENT-TO-DECK: Analyze the following content and create a complete, visually stunning presentation from it.

Design style: ${designStyle}
Slide count preference: ${slideCount === 'auto' ? 'Decide the optimal number based on content (usually 5-8)' : slideCount + ' slides'}

INSTRUCTIONS:
1. Read and understand the content thoroughly
2. Extract key themes, sections, and important points
3. Plan a logical slide flow: Title → Key sections → Conclusion/CTA
4. Create visually rich slides with decorative shapes, accent colors, and clean typography
5. Summarize dense text into concise bullet points or short phrases
6. Use the "generate_deck" action

CONTENT:
---
${content}
---`;

    try {
      const result = await ProcessAIPrompt(state.currentSlide, prompt);
      state.deck = result.deck;
      state.currentSlide = 0;
      state.selectedElement = null;

      state.aiMessages.push({
        role: 'assistant',
        content: `✨ ${result.message || `Generated "${state.deck.meta.title}" with ${state.deck.slides.length} slides from your content.`}`,
        reasoning: state.aiReasoning,
        showReasoning: true,
      });

      state.aiContentMode = false;
      renderAppShell();
    } catch (e) {
      console.error('AI Content Error:', e);
      state.aiMessages.push({
        role: 'assistant',
        content: `⚠️ Failed to generate from content: ${e}`,
        reasoning: state.aiReasoning,
        showReasoning: true,
      });
      state.aiContentMode = false;
      updatePropertiesOnly();
    } finally {
      state.aiLoading = false;
      state.aiReasoning = '';
      state.aiStreamingText = '';
      updatePropertiesOnly();
    }
  },

  async sendAIPrompt(promptText) {
    if (!promptText || state.aiLoading) return;
    state.aiMessages.push({ role: 'user', content: promptText });
    state.aiLoading = true;
    state.aiReasoning = '';
    state.aiStreamingText = '';
    updateAIMessagesOnly();

    // Parse @slide1, @slide2 etc. to target a specific slide
    let targetSlide = state.currentSlide;
    const slideTagMatch = promptText.match(/@slide\s*(\d+)/i);
    if (slideTagMatch) {
      const taggedIdx = parseInt(slideTagMatch[1], 10) - 1; // 1-based → 0-based
      if (taggedIdx >= 0 && taggedIdx < state.deck.slides.length) {
        targetSlide = taggedIdx;
      }
    }

    try {
      const result = await ProcessAIPrompt(targetSlide, promptText);
      state.deck = result.deck;

      let responseMsg = '';
      switch (result.action) {
        case 'generate_deck':
          state.currentSlide = 0;
          state.selectedElement = null;
          responseMsg = `✨ ${result.message || `Generated "${state.deck.meta.title}" with ${state.deck.slides.length} slides.`}`;
          break;
        case 'add_slide':
          state.currentSlide = state.deck.slides.length - 1;
          state.selectedElement = null;
          responseMsg = `✨ ${result.message || 'Added a new slide.'}`;
          break;
        case 'edit_slide':
          state.currentSlide = targetSlide;
          state.selectedElement = null;
          responseMsg = `✨ ${result.message || `Updated slide ${targetSlide + 1}.`}`;
          break;
        default:
          responseMsg = `✨ ${result.message || 'Done.'}`;
      }

      state.aiMessages.push({
        role: 'assistant',
        content: responseMsg,
        reasoning: state.aiReasoning,
        showReasoning: true,
      });
      renderAppShell();
    } catch (e) {
      console.error('AI Error:', e);
      state.aiMessages.push({
        role: 'assistant',
        content: `⚠️ Failed to process AI prompt: ${e}`,
        reasoning: state.aiReasoning,
        showReasoning: true,
      });
      updateAIMessagesOnly();
    } finally {
      state.aiLoading = false;
      state.aiReasoning = '';
      state.aiStreamingText = '';
      updateAIMessagesOnly();
    }
  },

  copyMsg(index, event) {
    const msg = state.aiMessages[index];
    if (!msg) return;
    const textToCopy = (msg.reasoning ? `[Model Reasoning]\n${msg.reasoning}\n\n[Message]\n` : '') + msg.content;
    navigator.clipboard.writeText(textToCopy).then(() => {
      const btn = event.currentTarget;
      if (btn) {
        const origHTML = btn.innerHTML;
        btn.innerHTML = `✓ Copied!`;
        setTimeout(() => { btn.innerHTML = origHTML; }, 1500);
      }
    }).catch(err => {
      console.error('Copy failed:', err);
    });
  },

  toggleMsgReasoning(index) {
    const msg = state.aiMessages[index];
    if (msg) {
      msg.showReasoning = msg.showReasoning === false ? true : false;
      updateAIMessagesOnly();
    }
  },

  submitAIPrompt() {
    const input = document.getElementById('aiInput');
    if (!input || !input.value.trim()) return;
    const text = input.value.trim();
    input.value = '';
    window.app.sendAIPrompt(text);
  },

  // Slide navigation
  selectSlide(index) {
    state.currentSlide = index;
    state.selectedElement = null;
    updateSidebarOnly();
    updateCanvasOnly();
  },

  async addSlide() {
    state.deck = await AddSlide(-1);
    state.currentSlide = state.deck.slides.length - 1;
    state.selectedElement = null;
    updateSidebarOnly();
    updateCanvasOnly();
  },

  async deleteSlide(index) {
    const idx = index ?? state.currentSlide;
    state.deck = await DeleteSlide(idx);
    if (state.currentSlide >= state.deck.slides.length) {
      state.currentSlide = state.deck.slides.length - 1;
    }
    state.selectedElement = null;
    state.contextMenu = null;
    updateSidebarOnly();
    updateCanvasOnly();
  },

  async duplicateSlide() {
    state.deck = await DuplicateSlide(state.currentSlide);
    state.currentSlide++;
    state.selectedElement = null;
    updateSidebarOnly();
    updateCanvasOnly();
  },

  // Elements
  async addElement(type) {
    state.deck = await AddElement(state.currentSlide, type);
    const slide = getCurrentSlide();
    state.selectedElement = slide.elements[slide.elements.length - 1].id;
    updateCanvasOnly();
  },

  selectElement(e, id) {
    e.stopPropagation();
    if (state.selectedElement !== id) {
      state.selectedElement = id;
      state.contextMenu = null;
      updateCanvasOnly();
    }
  },

  async deleteSelected() {
    if (!state.selectedElement) return;
    state.deck = await DeleteElement(state.currentSlide, state.selectedElement);
    state.selectedElement = null;
    updateCanvasOnly();
  },

  startDrag(e, id) {
    if (state.editingText) return;
    e.stopPropagation();
    const el = getCurrentSlide().elements?.find(el => el.id === id);
    if (!el) return;

    state.selectedElement = id;
    state.isDragging = true;

    const canvas = document.getElementById('slideCanvas');
    const rect = canvas.getBoundingClientRect();
    const scale = state.canvasScale;

    const clickCanvasX = (e.clientX - rect.left) / scale;
    const clickCanvasY = (e.clientY - rect.top) / scale;

    state.dragOffset = {
      x: clickCanvasX - el.position.x,
      y: clickCanvasY - el.position.y,
    };
    updateCanvasOnly();
  },

  startResize(e, id, handle) {
    e.stopPropagation();
    e.preventDefault();
    const el = getCurrentSlide().elements?.find(el => el.id === id);
    if (!el) return;

    state.selectedElement = id;
    state.isResizing = true;
    state.resizeHandle = handle;
    state.resizeStart = {
      x: e.clientX,
      y: e.clientY,
      origPos: { ...el.position },
    };
  },

  // Text editing
  startTextEdit(id) {
    state.editingText = true;
    state.selectedElement = id;
    updateCanvasOnly();
    setTimeout(() => {
      const dom = document.querySelector(`[data-element-id="${id}"] .element-text`);
      if (dom) {
        dom.focus();
        const range = document.createRange();
        const sel = window.getSelection();
        range.selectNodeContents(dom);
        range.collapse(false);
        sel.removeAllRanges();
        sel.addRange(range);
      }
    }, 50);
  },

  async finishTextEdit(id, newText) {
    if (!state.editingText) return;
    state.editingText = false;
    const el = getCurrentSlide().elements?.find(el => el.id === id);
    if (el) {
      if (newText !== undefined && newText !== null) {
        const trimmed = newText.trim();
        if (trimmed.length > 0) {
          el.content = trimmed;
        }
      }
      state.deck = await UpdateElement(state.currentSlide, el);
      updateThumbnailsDebounced();
      updateCanvasOnly();
    }
  },

  toggleShapeMenu() {
    state.shapeMenuOpen = !state.shapeMenuOpen;
    renderAppShell();
  },

  async addShape(shapeType) {
    state.shapeMenuOpen = false;
    state.deck = await AddShapeElement(state.currentSlide, shapeType);
    const slide = getCurrentSlide();
    state.selectedElement = slide.elements[slide.elements.length - 1].id;
    renderAppShell();
  },

  async stepFontSize(delta) {
    const el = getSelectedElement();
    if (!el || el.type !== 'text') return;
    const current = el.style?.fontSize || 24;
    const next = Math.max(12, Math.min(120, current + delta));
    window.app.updateStyle('fontSize', next);
  },

  async updateTextColor(color) {
    window.app.updateStyle('color', color);
  },

  async updateShapeColor(color) {
    window.app.updateStyle('bgColor', color);
  },

  async updateShapeType(shapeType) {
    const el = getSelectedElement();
    if (!el) return;
    el.shapeType = shapeType;
    state.deck = await UpdateElement(state.currentSlide, el);
    updateCanvasOnly();
  },

  // Property updates
  async updateStyle(key, value) {
    const el = getSelectedElement();
    if (!el) return;
    if (!el.style) el.style = {};
    el.style[key] = value;
    state.deck = await UpdateElement(state.currentSlide, el);
    updateCanvasOnly();
  },

  async updatePosition(key, value) {
    const el = getSelectedElement();
    if (!el) return;
    el.position[key] = value;
    state.deck = await UpdateElement(state.currentSlide, el);
    updateCanvasOnly();
  },

  async updateSlideBg(color) {
    state.deck = await UpdateSlideBg(state.currentSlide, color);
    updateCanvasOnly();
  },

  async updateTitle(title) {
    state.deck = await UpdateDeckMeta(title, state.deck.meta.author || '');
    const titleText = document.getElementById('deckTitleText');
    const statusText = document.getElementById('statusTitleText');
    if (titleText) titleText.innerText = title;
    if (statusText) statusText.innerText = title;
  },

  // Undo/Redo
  async undo() {
    state.deck = await Undo();
    state.selectedElement = null;
    updateCanvasOnly();
  },

  async redo() {
    state.deck = await Redo();
    state.selectedElement = null;
    updateCanvasOnly();
  },

  toggleFileMenu(e) {
    if (e) e.stopPropagation();
    state.fileMenuOpen = !state.fileMenuOpen;
    renderAppShell();
  },

  // File operations
  async newDeck() {
    state.fileMenuOpen = false;
    state.deck = await NewDeck();
    state.currentSlide = 0;
    state.selectedElement = null;
    state.filePath = '';
    renderAppShell();
  },

  async openFile() {
    state.fileMenuOpen = false;
    try {
      const deck = await OpenFile();
      if (deck && deck.slides && deck.slides.length > 0) {
        state.deck = deck;
        state.currentSlide = 0;
        state.selectedElement = null;
        state.filePath = await GetFilePath();
        renderAppShell();
      }
    } catch (e) {
      console.error('Open failed:', e);
    }
  },

  async saveFile() {
    state.fileMenuOpen = false;
    try {
      const savedPath = await SaveFile();
      if (savedPath) {
        state.filePath = savedPath;
        renderAppShell();
      }
    } catch (e) {
      console.error('Save failed:', e);
    }
  },

  async saveFileAs() {
    state.fileMenuOpen = false;
    try {
      const savedPath = await SaveFileAs();
      if (savedPath) {
        state.filePath = savedPath;
        renderAppShell();
      }
    } catch (e) {
      console.error('Save As failed:', e);
    }
  },

  // Present mode
  present() {
    state.presenting = true;
    renderAppShell();
  },

  exitPresent() {
    state.presenting = false;
    renderAppShell();
  },

  presentNext() {
    if (state.currentSlide < state.deck.slides.length - 1) {
      state.currentSlide++;
      const overlay = document.getElementById('presentOverlay');
      if (overlay) overlay.outerHTML = renderPresentMode();
    }
  },

  presentPrev() {
    if (state.currentSlide > 0) {
      state.currentSlide--;
      const overlay = document.getElementById('presentOverlay');
      if (overlay) overlay.outerHTML = renderPresentMode();
    }
  },

  // Command palette
  toggleCmdPalette() {
    state.cmdPaletteOpen = !state.cmdPaletteOpen;
    const container = document.getElementById('modalContainer');
    if (container) {
      container.innerHTML = state.cmdPaletteOpen ? renderCommandPalette() : '';
      if (state.cmdPaletteOpen) {
        setTimeout(() => document.getElementById('cmdInput')?.focus(), 50);
      }
    }
  },

  closeCmdPalette() {
    state.cmdPaletteOpen = false;
    const container = document.getElementById('modalContainer');
    if (container) container.innerHTML = '';
  },

  filterCommands(query) {
    const results = document.getElementById('cmdResults');
    if (!results) return;
    const filtered = commands.filter(c => c.name.toLowerCase().includes(query.toLowerCase()));
    results.innerHTML = filtered.map((cmd, i) => `
      <div class="cmd-palette-item ${i === 0 ? 'highlighted' : ''}" onclick="window.app.runCommand(${commands.indexOf(cmd)})">
        <span class="cmd-palette-item-text">${cmd.name}</span>
        ${cmd.shortcut ? `<span class="cmd-palette-item-shortcut">${cmd.shortcut}</span>` : ''}
      </div>
    `).join('');
  },

  runCommand(index) {
    window.app.closeCmdPalette();
    commands[index]?.action();
  },

  // Context menus
  showSlideContextMenu(e, index) {
    e.preventDefault();
    e.stopPropagation();
    state.currentSlide = index;
    state.contextMenu = {
      x: e.clientX,
      y: e.clientY,
      items: [
        { label: 'Duplicate Slide', action: () => window.app.duplicateSlide() },
        { label: 'Add Slide Below', action: () => window.app.addSlide() },
        { sep: true },
        { label: 'Delete Slide', danger: true, action: () => window.app.deleteSlide(index) },
      ],
    };
    const container = document.getElementById('modalContainer');
    if (container) container.innerHTML = renderContextMenu();
  },

  showElementContextMenu(e, id) {
    e.preventDefault();
    e.stopPropagation();
    state.selectedElement = id;
    state.contextMenu = {
      x: e.clientX,
      y: e.clientY,
      items: [
        { label: 'Duplicate', action: () => { /* TODO */ } },
        { label: 'Bring to Front', action: () => { /* TODO */ } },
        { label: 'Send to Back', action: () => { /* TODO */ } },
        { sep: true },
        { label: 'Delete', danger: true, action: () => window.app.deleteSelected() },
      ],
    };
    const container = document.getElementById('modalContainer');
    if (container) container.innerHTML = renderContextMenu();
  },

  runContextAction(index) {
    const item = state.contextMenu?.items[index];
    state.contextMenu = null;
    const container = document.getElementById('modalContainer');
    if (container) container.innerHTML = '';
    if (item?.action) item.action();
  },
};

// ─── Boot ──────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', init);
