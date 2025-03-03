import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from "@angular/core";
import { Store } from "@ngxs/store";
import { Project } from "app/model/project.model";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { Tab } from "app/shared/tabs/tabs.component";
import { ProjectV2State } from "app/store/project-v2.state";
import { Subscription } from "rxjs";

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
	project: Project;
	projectSubscriber: Subscription;

	constructor(
		private _cd: ChangeDetectorRef,
		private _store: Store
	) { }

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnInit(): void {
		this.projectSubscriber = this._store.select(ProjectV2State.current)
			.subscribe(p => {
				this.project = p;
				this.initTabs();
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
		}, <Tab>{
			title: 'Variables Sets',
			icon: 'font-colors',
			iconTheme: 'outline',
			key: 'variables',
		}, <Tab>{
			title: 'Concurrencies',
			icon: 'font-colors',
			iconTheme: 'outline',
			key: 'concurrencies',
		}];
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