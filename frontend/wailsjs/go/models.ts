export namespace engine {
	
	export class Style {
	    fontSize?: number;
	    fontWeight?: string;
	    fontFamily?: string;
	    color?: string;
	    bgColor?: string;
	    borderColor?: string;
	    borderWidth?: number;
	    borderRadius?: number;
	    opacity?: number;
	    textAlign?: string;
	    lineHeight?: number;
	
	    static createFrom(source: any = {}) {
	        return new Style(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fontSize = source["fontSize"];
	        this.fontWeight = source["fontWeight"];
	        this.fontFamily = source["fontFamily"];
	        this.color = source["color"];
	        this.bgColor = source["bgColor"];
	        this.borderColor = source["borderColor"];
	        this.borderWidth = source["borderWidth"];
	        this.borderRadius = source["borderRadius"];
	        this.opacity = source["opacity"];
	        this.textAlign = source["textAlign"];
	        this.lineHeight = source["lineHeight"];
	    }
	}
	export class Position {
	    x: number;
	    y: number;
	    w: number;
	    h: number;
	
	    static createFrom(source: any = {}) {
	        return new Position(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	        this.w = source["w"];
	        this.h = source["h"];
	    }
	}
	export class Element {
	    id: string;
	    type: string;
	    content: string;
	    position: Position;
	    style: Style;
	    zIndex: number;
	    shapeType?: string;
	    imageUrl?: string;
	
	    static createFrom(source: any = {}) {
	        return new Element(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.content = source["content"];
	        this.position = this.convertValues(source["position"], Position);
	        this.style = this.convertValues(source["style"], Style);
	        this.zIndex = source["zIndex"];
	        this.shapeType = source["shapeType"];
	        this.imageUrl = source["imageUrl"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Slide {
	    id: string;
	    layout?: string;
	    elements: Element[];
	    notes?: string;
	    bgColor?: string;
	
	    static createFrom(source: any = {}) {
	        return new Slide(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.layout = source["layout"];
	        this.elements = this.convertValues(source["elements"], Element);
	        this.notes = source["notes"];
	        this.bgColor = source["bgColor"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ThemeFonts {
	    heading: string;
	    body: string;
	
	    static createFrom(source: any = {}) {
	        return new ThemeFonts(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.heading = source["heading"];
	        this.body = source["body"];
	    }
	}
	export class ThemeColors {
	    primary: string;
	    secondary: string;
	    bg: string;
	    surface: string;
	    text: string;
	    textMuted: string;
	
	    static createFrom(source: any = {}) {
	        return new ThemeColors(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.primary = source["primary"];
	        this.secondary = source["secondary"];
	        this.bg = source["bg"];
	        this.surface = source["surface"];
	        this.text = source["text"];
	        this.textMuted = source["textMuted"];
	    }
	}
	export class Theme {
	    name: string;
	    colors: ThemeColors;
	    fonts: ThemeFonts;
	
	    static createFrom(source: any = {}) {
	        return new Theme(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.colors = this.convertValues(source["colors"], ThemeColors);
	        this.fonts = this.convertValues(source["fonts"], ThemeFonts);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DeckMeta {
	    title: string;
	    author: string;
	    created: string;
	    modified: string;
	
	    static createFrom(source: any = {}) {
	        return new DeckMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.author = source["author"];
	        this.created = source["created"];
	        this.modified = source["modified"];
	    }
	}
	export class Deck {
	    version: string;
	    meta: DeckMeta;
	    theme: Theme;
	    slides: Slide[];
	
	    static createFrom(source: any = {}) {
	        return new Deck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.meta = this.convertValues(source["meta"], DeckMeta);
	        this.theme = this.convertValues(source["theme"], Theme);
	        this.slides = this.convertValues(source["slides"], Slide);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	export class RecentDeckItem {
	    path: string;
	    title: string;
	    modified: string;
	
	    static createFrom(source: any = {}) {
	        return new RecentDeckItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.title = source["title"];
	        this.modified = source["modified"];
	    }
	}
	
	
	
	

}

export namespace main {
	
	export class AIResult {
	    action: string;
	    deck: engine.Deck;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new AIResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.deck = this.convertValues(source["deck"], engine.Deck);
	        this.message = source["message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

