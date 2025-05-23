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

