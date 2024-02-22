import { ChangeDetectionStrategy, Component, Input, OnInit } from "@angular/core";
import { NavigationEnd, Router } from "@angular/router";
import { Store } from "@ngxs/store";
import { Project } from "app/model/project.model";
import { filter } from "rxjs";
import * as actionNavigation from 'app/store/navigation.action';
import { NavigationState } from "app/store/navigation.state";

@Component({
	selector: 'app-projectv2-activity-bar',
	templateUrl: './activity-bar.html',
	styleUrls: ['./activity-bar.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectV2ActivityBarComponent implements OnInit {
	@Input() project: Project;

	constructor(
		private _store: Store,
		private _router: Router
	) { }

	ngOnInit(): void {
		this._router.events.pipe(
			filter(e => e instanceof NavigationEnd),
		).forEach((e: NavigationEnd) => {
			if (e.url.startsWith(`/projectv2/${this.project.key}/run/`)) {
				this._store.dispatch(new actionNavigation.SetActivityLastRoute({ projectKey: this.project.key, activityKey: 'run', route: e.url }));
			} else if (e.url.startsWith(`/projectv2/${this.project.key}/run`)) {
				this._store.dispatch(new actionNavigation.SetActivityRunLastFilters({ projectKey: this.project.key, route: e.url }));
				this._store.dispatch(new actionNavigation.SetActivityLastRoute({ projectKey: this.project.key, activityKey: 'run', route: e.url }));
			} else if (e.url.startsWith(`/projectv2/${this.project.key}/explore`)) {
				this._store.dispatch(new actionNavigation.SetActivityLastRoute({ projectKey: this.project.key, activityKey: 'explore', route: e.url }));
			}
		});
	}

	clickActivity(event: Event, activityKey: string): void {
		const lastRoute = this._store.selectSnapshot(NavigationState.selectActivityLastRoute(this.project.key, activityKey));
		if (lastRoute) {
			if (activityKey == 'run') {
				const lastFilters = this._store.selectSnapshot(NavigationState.selectActivityRunLastFilters(this.project.key));
				if (lastFilters && this._router.isActive(lastRoute, { paths: 'exact', queryParams: 'exact', fragment: 'ignored', matrixParams: 'ignored' })) {
					this._router.navigateByUrl(lastFilters);
				} else {
					this._router.navigateByUrl(lastRoute);
				}
			} else {
				this._router.navigateByUrl(lastRoute);
			}
			event.stopPropagation();
			event.preventDefault();
		}
	}
}