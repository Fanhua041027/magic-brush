export namespace auth {
	
	export class User {
	    id: string;
	    username: string;
	    passwordHash: string;
	    role: string;
	    enabled: boolean;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    lastLoginAt?: any;
	    totalSeconds: number;
	    mustChangePassword?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new User(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.username = source["username"];
	        this.passwordHash = source["passwordHash"];
	        this.role = source["role"];
	        this.enabled = source["enabled"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.lastLoginAt = this.convertValues(source["lastLoginAt"], null);
	        this.totalSeconds = source["totalSeconds"];
	        this.mustChangePassword = source["mustChangePassword"];
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

export namespace config {
	
	export class Config {
	    apiKey?: string;
	    baseURL?: string;
	    model?: string;
	    screenshotApiKey?: string;
	    screenshotBaseUrl?: string;
	    screenshotModel?: string;
	    prompt?: string;
	    domainId?: string;
	    opacity?: number;
	    noCompression?: boolean;
	    compressionQuality?: number;
	    sharpening?: number;
	    grayscale?: boolean;
	    keepContext?: boolean;
	    interruptThinking?: boolean;
	    screenshotMode?: string;
	    resumePath?: string;
	    resumeContent?: string;
	    shortcuts?: Record<string, shortcut.KeyBinding>;
	    assistantModel?: string;
	    windowWidth?: number;
	    windowHeight?: number;
	    theme?: string;
	    sttModel?: string;
	    sttDevice?: string;
	    sttLanguage?: string;
	    sttSensitivity?: number;
	    sttService?: string;
	    answerLength?: string;
	    answerStyle?: string;
	    answerStructure?: string;
	    answerDuration?: string;
	    includeTechnicalDetails?: boolean;
	    kbPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiKey = source["apiKey"];
	        this.baseURL = source["baseURL"];
	        this.model = source["model"];
	        this.screenshotApiKey = source["screenshotApiKey"];
	        this.screenshotBaseUrl = source["screenshotBaseUrl"];
	        this.screenshotModel = source["screenshotModel"];
	        this.prompt = source["prompt"];
	        this.domainId = source["domainId"];
	        this.opacity = source["opacity"];
	        this.noCompression = source["noCompression"];
	        this.compressionQuality = source["compressionQuality"];
	        this.sharpening = source["sharpening"];
	        this.grayscale = source["grayscale"];
	        this.keepContext = source["keepContext"];
	        this.interruptThinking = source["interruptThinking"];
	        this.screenshotMode = source["screenshotMode"];
	        this.resumePath = source["resumePath"];
	        this.resumeContent = source["resumeContent"];
	        this.shortcuts = this.convertValues(source["shortcuts"], shortcut.KeyBinding, true);
	        this.assistantModel = source["assistantModel"];
	        this.windowWidth = source["windowWidth"];
	        this.windowHeight = source["windowHeight"];
	        this.theme = source["theme"];
	        this.sttModel = source["sttModel"];
	        this.sttDevice = source["sttDevice"];
	        this.sttLanguage = source["sttLanguage"];
	        this.sttSensitivity = source["sttSensitivity"];
	        this.sttService = source["sttService"];
	        this.answerLength = source["answerLength"];
	        this.answerStyle = source["answerStyle"];
	        this.answerStructure = source["answerStructure"];
	        this.answerDuration = source["answerDuration"];
	        this.includeTechnicalDetails = source["includeTechnicalDetails"];
	        this.kbPath = source["kbPath"];
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

export namespace domain {
	
	export class DomainItem {
	    id: string;
	    label: string;
	    icon: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new DomainItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.icon = source["icon"];
	        this.description = source["description"];
	    }
	}
	export class Category {
	    id: string;
	    label: string;
	    items: DomainItem[];
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.items = this.convertValues(source["items"], DomainItem);
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

export namespace screen {
	
	export class PreviewResult {
	    imgBytes: number[];
	    base64: string;
	    size: string;
	
	    static createFrom(source: any = {}) {
	        return new PreviewResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.imgBytes = source["imgBytes"];
	        this.base64 = source["base64"];
	        this.size = source["size"];
	    }
	}

}

export namespace shortcut {
	
	export class KeyBinding {
	    vkCode: string;
	    keyName: string;
	
	    static createFrom(source: any = {}) {
	        return new KeyBinding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vkCode = source["vkCode"];
	        this.keyName = source["keyName"];
	    }
	}

}

