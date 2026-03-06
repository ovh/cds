import { ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, OnInit } from "@angular/core";
import { Store } from "@ngxs/store";
import { Project } from "app/model/project.model";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { PreferencesState } from "app/store/preferences.state";
import { ProjectV2State } from "app/store/project-v2.state";
import { NzDrawerService } from "ng-zorro-antd/drawer";
import { Subscription } from "rxjs";
import { ProjectV2RunStartComponent, ProjectV2RunStartComponentParams } from "../run-start/run-start.component";
import { ProjectV2RepositoryAddComponent, ProjectV2RepositoryAddComponentParams } from "./repository-add/repository-add.component";

@Component({
	standalone: false,
	selector: 'app-projectv2-explore-overview',
	templateUrl: './explore-overview.html',
	styleUrls: ['./explore-overview.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2ExploreOverviewComponent implements OnInit {

	project: Project;
	themeSubscription: Subscription;
	isNight: boolean;

	private _store = inject(Store);
	private _cd = inject(ChangeDetectorRef);
	private _drawerService = inject(NzDrawerService);

	constructor() {
		this.project = this._store.selectSnapshot(ProjectV2State.current);
	}

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnInit() {
		this.themeSubscription = this._store.select(PreferencesState.theme)
			.subscribe(t => {
				this.isNight = t === 'night';
				this._cd.markForCheck();
			});
	}

	openRunStartDrawer(): void {
		const drawerRef = this._drawerService.create<ProjectV2RunStartComponent, { value: string }, string>({
			nzTitle: 'Start new Workflow Run',
			nzContent: ProjectV2RunStartComponent,
			nzContentParams: {
				params: <ProjectV2RunStartComponentParams>{}
			},
			nzSize: 'large',
			nzBodyStyle: { 'padding': '0' }
		});
		drawerRef.afterClose.subscribe(data => { });
	}

	openRepositoryAddDrawer(): void {
		const drawerRef = this._drawerService.create<ProjectV2RepositoryAddComponent, { value: string }, string>({
			nzTitle: 'Add a new Repository',
			nzContent: ProjectV2RepositoryAddComponent,
			nzContentParams: {
				params: <ProjectV2RepositoryAddComponentParams>{}
			},
			nzSize: 'large'
		});
		drawerRef.afterClose.subscribe(data => { });
	}

}
