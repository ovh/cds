import { ChangeDetectionStrategy, Component, OnInit } from "@angular/core";
import { Store } from "@ngxs/store";
import { Project } from "app/model/project.model";
import * as actionPreferences from 'app/store/preferences.action';
import { PreferencesState } from "app/store/preferences.state";
import { ProjectState } from "app/store/project.state";

@Component({
	selector: 'app-projectv2-explore',
	templateUrl: './explore.html',
	styleUrls: ['./explore.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectV2ExploreComponent implements OnInit {
	static PANEL_KEY = 'project-v2-explore-sidebar';

	project: Project;
	panelSize: number | string;

	constructor(
		private _store: Store
	) {
		this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
	}

	ngOnInit(): void {
		this.panelSize = this._store.selectSnapshot(PreferencesState.panelSize(ProjectV2ExploreComponent.PANEL_KEY)) ?? '15%';
	}

	panelStartResize(): void {
		this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: true }));
	}

	panelEndResize(size: number): void {
		this._store.dispatch(new actionPreferences.SavePanelSize({ panelKey: ProjectV2ExploreComponent.PANEL_KEY, size: size }));
		this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: false }));
	}
}