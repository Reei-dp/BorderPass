export namespace config {
	
	export class Config {
	    role: string;
	    position: string;
	    address: string;
	    screen_width: number;
	    screen_height: number;
	    edge_threshold: number;
	    mouse_device: string;
	    keyboard_device: string;
	    enable_clipboard: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.position = source["position"];
	        this.address = source["address"];
	        this.screen_width = source["screen_width"];
	        this.screen_height = source["screen_height"];
	        this.edge_threshold = source["edge_threshold"];
	        this.mouse_device = source["mouse_device"];
	        this.keyboard_device = source["keyboard_device"];
	        this.enable_clipboard = source["enable_clipboard"];
	    }
	}

}

export namespace main {
	
	export class TelemetryData {
	    isRunning: boolean;
	    role: string;
	    status: string;
	    remoteAddr: string;
	    activeControl: string;
	    packetsSent: number;
	    packetsRecv: number;
	    clipboardEvents: number;
	    lastError: string;
	
	    static createFrom(source: any = {}) {
	        return new TelemetryData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isRunning = source["isRunning"];
	        this.role = source["role"];
	        this.status = source["status"];
	        this.remoteAddr = source["remoteAddr"];
	        this.activeControl = source["activeControl"];
	        this.packetsSent = source["packetsSent"];
	        this.packetsRecv = source["packetsRecv"];
	        this.clipboardEvents = source["clipboardEvents"];
	        this.lastError = source["lastError"];
	    }
	}

}

