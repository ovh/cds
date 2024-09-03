export class CDNLogLink {
	item_type: string;
	api_ref: string;
}

export class CDNLogsLines {
	api_ref: string
	lines_count: number
}

export class CDNLogLinks {
	datas: Array<CDNLogLink>;
}

export class CDNLinesResponse {
	totalCount: number;
	lines: Array<CDNLine>;
}

export class CDNLine {
	number: number;
	value: string;
	api_ref_hash: string;
	since: number; // the count of milliseconds since job start

	// properties used by ui only
	extra: Array<string>;
}

export class CDNStreamFilter {
	item_type: string;
	job_run_id: string;
}