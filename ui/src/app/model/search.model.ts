export enum SearchResultType {
	Project = "project",
	WorkflowV2 = "workflow-v2",
	Workflow = "workflow"
}

export class SearchResult {
	type: SearchResultType;
	id: string;
	label: string;
	variants: Array<string>;
}

export class SearchResponse {
	totalCount: number;
	results: Array<SearchResult>;
}