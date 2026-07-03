export namespace main {
	
	export class DbItem {
	    id: string;
	    title: string;
	    content: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new DbItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class DiscoveredDevice {
	    ip: string;
	    port: number;
	    deviceName: string;
	
	    static createFrom(source: any = {}) {
	        return new DiscoveredDevice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.port = source["port"];
	        this.deviceName = source["deviceName"];
	    }
	}

}

