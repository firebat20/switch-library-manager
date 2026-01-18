export namespace db {
	
	export class TitleAttributes {
	    id: string;
	    name?: string;
	    version?: string;
	    region?: string;
	    releaseDate?: number;
	    ParsedReleaseDate: string;
	    publisher?: string;
	    iconUrl?: string;
	    screenshots?: string[];
	    bannerUrl?: string;
	    description?: string;
	    size?: number;
	    isDemo?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TitleAttributes(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.region = source["region"];
	        this.releaseDate = source["releaseDate"];
	        this.ParsedReleaseDate = source["ParsedReleaseDate"];
	        this.publisher = source["publisher"];
	        this.iconUrl = source["iconUrl"];
	        this.screenshots = source["screenshots"];
	        this.bannerUrl = source["bannerUrl"];
	        this.description = source["description"];
	        this.size = source["size"];
	        this.isDemo = source["isDemo"];
	    }
	}

}

export namespace main {
	
	export class LibraryTemplateData {
	    id: number;
	    name: string;
	    version: string;
	    dlc: string;
	    titleId: string;
	    path: string;
	    icon: string;
	    update: number;
	    region: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new LibraryTemplateData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.dlc = source["dlc"];
	        this.titleId = source["titleId"];
	        this.path = source["path"];
	        this.icon = source["icon"];
	        this.update = source["update"];
	        this.region = source["region"];
	        this.type = source["type"];
	    }
	}
	export class Pair {
	    key: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new Pair(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	    }
	}
	export class LocalLibraryData {
	    library_data: LibraryTemplateData[];
	    issues: Pair[];
	    num_files: number;
	
	    static createFrom(source: any = {}) {
	        return new LocalLibraryData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.library_data = this.convertValues(source["library_data"], LibraryTemplateData);
	        this.issues = this.convertValues(source["issues"], Pair);
	        this.num_files = source["num_files"];
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
	
	export class SwitchTitle {
	    name: string;
	    titleId: string;
	    icon: string;
	    region: string;
	    release_date: string;
	
	    static createFrom(source: any = {}) {
	        return new SwitchTitle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.titleId = source["titleId"];
	        this.icon = source["icon"];
	        this.region = source["region"];
	        this.release_date = source["release_date"];
	    }
	}

}

export namespace process {
	
	export class IncompleteTitle {
	    Attributes: db.TitleAttributes;
	    Meta?: switchfs.ContentMetaAttributes;
	    local_update: number;
	    latest_update: number;
	    latest_update_date: string;
	    missing_dlc: string[];
	
	    static createFrom(source: any = {}) {
	        return new IncompleteTitle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Attributes = this.convertValues(source["Attributes"], db.TitleAttributes);
	        this.Meta = this.convertValues(source["Meta"], switchfs.ContentMetaAttributes);
	        this.local_update = source["local_update"];
	        this.latest_update = source["latest_update"];
	        this.latest_update_date = source["latest_update_date"];
	        this.missing_dlc = source["missing_dlc"];
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

export namespace switchfs {
	
	export class Content {
	    Text: string;
	    Type: string;
	    ID: string;
	    Size: string;
	    Hash: string;
	    KeyGeneration: string;
	
	    static createFrom(source: any = {}) {
	        return new Content(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Text = source["Text"];
	        this.Type = source["Type"];
	        this.ID = source["ID"];
	        this.Size = source["Size"];
	        this.Hash = source["Hash"];
	        this.KeyGeneration = source["KeyGeneration"];
	    }
	}
	export class NacpTitle {
	    Language: number;
	    Title: string;
	
	    static createFrom(source: any = {}) {
	        return new NacpTitle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Language = source["Language"];
	        this.Title = source["Title"];
	    }
	}
	export class Nacp {
	    TitleName: {[key: string]: NacpTitle};
	    Isbn: string;
	    DisplayVersion: string;
	    SupportedLanguageFlag: number;
	
	    static createFrom(source: any = {}) {
	        return new Nacp(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.TitleName = this.convertValues(source["TitleName"], NacpTitle, true);
	        this.Isbn = source["Isbn"];
	        this.DisplayVersion = source["DisplayVersion"];
	        this.SupportedLanguageFlag = source["SupportedLanguageFlag"];
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
	export class ContentMetaAttributes {
	    title_id: string;
	    version: number;
	    type: string;
	    Contents: {[key: string]: Content};
	    Ncap?: Nacp;
	
	    static createFrom(source: any = {}) {
	        return new ContentMetaAttributes(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title_id = source["title_id"];
	        this.version = source["version"];
	        this.type = source["type"];
	        this.Contents = this.convertValues(source["Contents"], Content, true);
	        this.Ncap = this.convertValues(source["Ncap"], Nacp);
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

