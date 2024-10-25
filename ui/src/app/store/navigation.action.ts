export class SetActivityLastRoute {
	static readonly type = '[Navigation] Set activity last route';
	constructor(public payload: { projectKey: string, activityKey: string, route: string }) { }
}

export class SetActivityRunLastFilters {
	static readonly type = '[Navigation] Set activity run last filters';
	constructor(public payload: { projectKey: string, route: string }) { }
}