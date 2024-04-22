import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, OnInit, SimpleChanges } from "@angular/core";
import { NavigationEnd, Router } from "@angular/router";
import { Store } from "@ngxs/store";
import { Project } from "app/model/project.model";
import { filter } from "rxjs";
import * as actionNavigation from 'app/store/navigation.action';
import { NavigationState } from "app/store/navigation.state";
import { FeatureNames, FeatureService } from "app/service/feature/feature.service";

@Component({
	selector: 'app-project-activity-bar',
	templateUrl: './activity-bar.html',
	styleUrls: ['./activity-bar.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectActivityBarComponent implements OnInit, OnChanges {
	@Input() project: Project;

	homeActive: boolean;
	v2Enabled: boolean;

	constructor(
		private _store: Store,
		private _router: Router,
		private _cd: ChangeDetectorRef,
		private _featureService: FeatureService
	) { }

	ngOnInit(): void {
		this._router.events.pipe(
			filter(e => e instanceof NavigationEnd),
		).forEach((e: NavigationEnd) => {
			this.updateRoute(e.url);
		});
		this.updateRoute(this._router.routerState.snapshot.url);
		this._featureService.isEnabled(FeatureNames.AllAsCode, { project_key: this.project.key }).subscribe(f => {
			this.v2Enabled = f.enabled;
			this._cd.markForCheck();
		});
	}

	ngOnChanges(changes: SimpleChanges): void {
		this._featureService.isEnabled(FeatureNames.AllAsCode, { project_key: this.project.key }).subscribe(f => {
			this.v2Enabled = f.enabled;
			this._cd.markForCheck();
		});
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