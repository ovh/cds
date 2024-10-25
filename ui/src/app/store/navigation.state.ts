import { Injectable } from "@angular/core";
import { Action, State, StateContext, createSelector } from "@ngxs/store";
import * as actionNavigation from './navigation.action';

export class NavigationStateModel {
	activitiesLastRoute: { [projectKey: string]: { [activityKey: string]: string } }
	activityRunLastFilters: { [projectKey: string]: string }
}

@State<NavigationStateModel>({
	name: 'navigation',
	defaults: {
		activitiesLastRoute: {},
		activityRunLastFilters: {}
	}
})
@Injectable()
export class NavigationState {
	constructor() { }

	static selectActivityLastRoute(projectKey: string, activityKey: string) {
		return createSelector(
			[NavigationState],
			(state: NavigationStateModel): string => {
				if (!state.activitiesLastRoute[projectKey]) {
					return null;
				}
				return state.activitiesLastRoute[projectKey][activityKey] ?? null;
			}
		);
	}

	static selectActivityRunLastFilters(projectKey: string) {
		return createSelector(
			[NavigationState],
			(state: NavigationStateModel): string => {
				if (!state.activityRunLastFilters[projectKey]) {
					return null;
				}
				return state.activityRunLastFilters[projectKey] ?? null;
			}
		);
	}

	@Action(actionNavigation.SetActivityLastRoute)
	setActivityLastRoute(ctx: StateContext<NavigationStateModel>, action: actionNavigation.SetActivityLastRoute) {
		const state = ctx.getState();

		let projects = {
			...state.activitiesLastRoute
		};
		if (!projects[action.payload.projectKey]) {
			projects[action.payload.projectKey] = {};
		}
		let activities = {
			...projects[action.payload.projectKey]
		};
		activities[action.payload.activityKey] = action.payload.route;
		projects[action.payload.projectKey] = activities;

		ctx.setState({
			...state,
			activitiesLastRoute: projects
		});
	}

	@Action(actionNavigation.SetActivityRunLastFilters)
	setActivityRunLastFilters(ctx: StateContext<NavigationStateModel>, action: actionNavigation.SetActivityRunLastFilters) {
		const state = ctx.getState();

		let projects = {
			...state.activityRunLastFilters
		};
		projects[action.payload.projectKey] = action.payload.route;

		ctx.setState({
			...state,
			activityRunLastFilters: projects
		});
	}
}

