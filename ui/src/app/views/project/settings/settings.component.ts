import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from "@angular/core";
import { ActivatedRoute } from "@angular/router";
import { Store } from "@ngxs/store";
import { Project } from "app/model/project.model";
import { FeatureNames, FeatureService } from "app/service/feature/feature.service";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { Tab } from "app/shared/tabs/tabs.component";
import { AddFeatureResult, FeaturePayload } from "app/store/feature.action";
import { ProjectState, ProjectStateModel } from "app/store/project.state";
import { cloneDeep } from "lodash-es";
import { Subscription, filter } from "rxjs";

@Component({
	selector: 'app-project-settings',
	templateUrl: './settings.html',
	styleUrls: ['./settings.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectSettingsComponent implements OnInit, OnDestroy {

	tabs: Array<Tab>;
	selectedTab: Tab;
	ascodeEnabled: boolean = false;
	project: Project;
	projectSubscriber: Subscription;

	constructor(
		private _cd: ChangeDetectorRef,
		private _featureService: FeatureService,
		private _store: Store,
		private _activatedRoute: ActivatedRoute
	) { }

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnInit(): void {
		this.projectSubscriber = this._store.select(ProjectState)
			.pipe(filter((projState: ProjectStateModel) => projState && projState.project &&
				projState.project.key !== null && !projState.project.externalChange &&
				this._activatedRoute.snapshot.parent.params['key'] === projState.project.key))
			.subscribe((projState: ProjectStateModel) => {
				let proj = cloneDeep(projState.project); // TODO: to delete when all will be in store, here it is useful to skip readonly

				if (!this.project || this.project.key !== proj?.key) {
					let data = { 'project_key': proj.key }
					this._featureService.isEnabled(FeatureNames.AllAsCode, data).subscribe(f => {
						this.ascodeEnabled = f.enabled;
						this._store.dispatch(new AddFeatureResult(<FeaturePayload>{
							key: f.name,
							result: {
								paramString: JSON.stringify(data),
								enabled: f.enabled,
								exists: f.exists
							}
						}));
						this.initTabs();
					});
				}
				this.project = proj;
				this.initTabs();
				if (this.project.integrations) {
					this.project.integrations.forEach(integ => {
						if (!integ.model.default_config) {
							return;
						}
						let keys = Object.keys(integ.model.default_config);
						if (keys) {
							keys.forEach(k => {
								if (!integ.config) {
									integ.config = {};
								}
								if (!integ.config[k]) {
									integ.config[k] = integ.model.default_config[k];
								}
							});
						}
					});
				}

				this._cd.markForCheck();
			});
	}

	initTabs(): void {
		let tabs = [<Tab>{
			title: 'Keys',
			icon: 'lock',
			iconTheme: 'outline',
			key: 'keys'
		}, <Tab>{
			title: 'Integrations',
			icon: 'usb',
			iconTheme: 'outline',
			key: 'integrations'
		}];
		if (this.ascodeEnabled) {
			tabs = [<Tab>{
				title: 'Variables Sets',
				icon: 'font-colors',
				iconTheme: 'outline',
				key: 'variables',
			}].concat(tabs);
		}
		if (this.project?.permissions?.writable) {
			tabs.push(<Tab>{
				title: 'Advanced',
				icon: 'setting',
				iconTheme: 'fill',
				key: 'advanced'
			});
		}
		this.tabs = tabs;
	}

	selectTab(tab: Tab): void {
		this.selectedTab = tab;
		this._cd.markForCheck();
	}
}