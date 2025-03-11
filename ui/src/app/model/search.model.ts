export enum SearchResultType {
	Project = "project",
	Workflow = "workflow",
	WorkflowLegacy = "workflow-legacy"
}

export class SearchResult {
	type: SearchResultType;
	id: string;
	label: string;
	description: string;
	variants: Array<string>;
}

export class SearchResponse {
	totalCount: number;
	results: Array<SearchResult>;
}