export namespace models {
	
	export class SimData {
	    CLI: string;
	    MSISDN: string;
	    SIM_STATUS_CHANGE: string;
	    RATE_PLAN_FULL_NAME: string;
	    CUSTOMER_LABEL_1: string;
	    CUSTOMER_LABEL_2: string;
	    CUSTOMER_LABEL_3: string;
	    SIM_SWAP: string;
	    IMSI: string;
	    IMEI: string;
	    APN_NAME: string;
	    IP1: string;
	    MONTHLY_USAGE_MB: string;
	    ALLOCATED_MB: string;
	    LAST_SESSION_TIME: string;
	    IN_SESSION: string;
	
	    static createFrom(source: any = {}) {
	        return new SimData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CLI = source["CLI"];
	        this.MSISDN = source["MSISDN"];
	        this.SIM_STATUS_CHANGE = source["SIM_STATUS_CHANGE"];
	        this.RATE_PLAN_FULL_NAME = source["RATE_PLAN_FULL_NAME"];
	        this.CUSTOMER_LABEL_1 = source["CUSTOMER_LABEL_1"];
	        this.CUSTOMER_LABEL_2 = source["CUSTOMER_LABEL_2"];
	        this.CUSTOMER_LABEL_3 = source["CUSTOMER_LABEL_3"];
	        this.SIM_SWAP = source["SIM_SWAP"];
	        this.IMSI = source["IMSI"];
	        this.IMEI = source["IMEI"];
	        this.APN_NAME = source["APN_NAME"];
	        this.IP1 = source["IP1"];
	        this.MONTHLY_USAGE_MB = source["MONTHLY_USAGE_MB"];
	        this.ALLOCATED_MB = source["ALLOCATED_MB"];
	        this.LAST_SESSION_TIME = source["LAST_SESSION_TIME"];
	        this.IN_SESSION = source["IN_SESSION"];
	    }
	}

}

