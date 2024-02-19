export class SetActivityLastRoute {
	static readonly type = '[Navigation] Set activity last route';
	constructor(public payload: { projectKey: string, activityKey: string, route: string }) { }
}