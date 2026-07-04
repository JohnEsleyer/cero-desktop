export namespace main {
	
	export class DbPage {
	    id: string;
	    parent_id?: string;
	    title: string;
	    content: string;
	    emoji: string;
	    created_at: string;
	    updated_at: string;
	    is_archived: number;
	    sort_order: number;
	    revision: number;
	
	    static createFrom(source: any = {}) {
	        return new DbPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.parent_id = source["parent_id"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.emoji = source["emoji"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	        this.is_archived = source["is_archived"];
	        this.sort_order = source["sort_order"];
	        this.revision = source["revision"];
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

