export enum EventV2Type {
	EventRunCrafted = "RunCrafted",
	EventRunBuilding = "RunBuilding",
	EventRunEnded = "RunEnded",
	EventRunRestart = "RunRestart"
}

export class FullEventV2 {
	id: string;
	type: EventV2Type;
	payload: any;
	project_key: string;
	vcs_name: string;
	repository: string;
	workflow: string;
	workflow_run_id: string;
	run_job_id: string;
	run_number: number;
	run_attempt: number;
	region: string;
	hatchery: string;
	model_type: string;
	job_id: string;
	status: string;
	user_id: string;
	username: string;
	run_result: string;
	entity: string;
	organization: string;
	permission: string;
	plugin: string;
	gpg_key: string;
	integration_model: string;
	integration: string;
	key_name: string;
	key_type: string;
	variable: string;
	notification: string;
	variable_set: string;
	item: string;
	timestamp: string;
}