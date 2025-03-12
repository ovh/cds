import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from "@angular/core";
import { NavigationEnd, Router } from "@angular/router";
import { Store } from "@ngxs/store";
import { Project } from "app/model/project.model";
import { filter, Subscription } from "rxjs";
import * as actionNavigation from 'app/store/navigation.action';
import { NavigationState } from "app/store/navigation.state";
import { ProjectState } from "app/store/project.state";
import { ProjectV2State } from "app/store/project-v2.state";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";

@Component({
	selector: 'app-project-activity-bar',
	templateUrl: './activity-bar.html',
	styleUrls: ['./activity-bar.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectActivityBarComponent implements OnInit, OnDestroy {
	projectv1: Project;
	projectv2: Project;
	project: Project;
	homeActive: boolean;
	projectV1Sub: Subscription;
	projectV2Sub: Subscription;

	constructor(
		private _store: Store,
		private _router: Router,
		private _cd: ChangeDetectorRef
	) {
		this.projectv1 = this._store.selectSnapshot(ProjectState.projectSnapshot);
		this.project = this.projectv1;
		this.projectv2 = this._store.selectSnapshot(ProjectV2State.current);
		if (!this.project) { this.project = this.projectv2; }
	}

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnInit(): void {
		this._router.events.pipe(
			filter(e => e instanceof NavigationEnd),
		).forEach((e: NavigationEnd) => {
			this.updateRoute(e.url);
		});

		this.projectV1Sub = this._store.select(ProjectState.projectSnapshot).subscribe(p => {
			this.projectv1 = p;
			if (p && p.key) { this.project = p; }
		});

		this.projectV2Sub = this._store.select(ProjectV2State.current).subscribe(p => {
			this.projectv2 = p;
			if (!p) { return; }
			if (!this.project || this.project.key !== this.projectv2.key) { this.project = this.projectv2; }
		});

		this.updateRoute(this._router.routerState.snapshot.url);
	}

	updateRoute(url: string): void {
		this.homeActive = false;
		if (url.startsWith(`/project/${this.project.key}/run/`)) {
			this._store.dispatch(new actionNavigation.SetActivityLastRoute({ projectKey: this.project.key, activityKey: 'run', route: url }));
		} else if (url.startsWith(`/project/${this.project.key}/run`)) {
			this._store.dispatch(new actionNavigation.SetActivityRunLastFilters({ projectKey: this.project.key, route: url }));
			this._store.dispatch(new actionNavigation.SetActivityLastRoute({ projectKey: this.project.key, activityKey: 'run', route: url }));
		} else if (url.startsWith(`/project/${this.project.key}/explore`)) {
			this._store.dispatch(new actionNavigation.SetActivityLastRoute({ projectKey: this.project.key, activityKey: 'explore', route: url }));
		} else if (!url.startsWith(`/project/${this.project.key}/settings`)) {
			this.homeActive = true;
		}
		this._cd.markForCheck();
	}

	clickActivity(event: Event, activityKey: string): void {
		let stopPropagation = false;
		const lastRoute = this._store.selectSnapshot(NavigationState.selectActivityLastRoute(this.project.key, activityKey));
		if (lastRoute) {
			if (activityKey == 'run') {
				const lastFilters = this._store.selectSnapshot(NavigationState.selectActivityRunLastFilters(this.project.key));
				if (this._router.isActive(lastRoute, { paths: 'exact', queryParams: 'exact', fragment: 'ignored', matrixParams: 'ignored' })) {
					if (lastFilters) {
						this._router.navigateByUrl(lastFilters);
						stopPropagation = true;
					}
				} else {
					this._router.navigateByUrl(lastRoute);
					stopPropagation = true;
				}
			} else {
				this._router.navigateByUrl(lastRoute);
				stopPropagation = true;
			}
		}
		if (stopPropagation) {
			event.stopPropagation();
			event.preventDefault();
		}
	}
}