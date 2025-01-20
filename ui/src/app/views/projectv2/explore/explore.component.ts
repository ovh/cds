import { ChangeDetectionStrategy, Component, OnInit } from "@angular/core";
import { Store } from "@ngxs/store";
import { Project } from "app/model/project.model";
import * as actionPreferences from 'app/store/preferences.action';
import { PreferencesState } from "app/store/preferences.state";
import { ProjectV2State } from "app/store/project-v2.state";

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
		this.project = this._store.selectSnapshot(ProjectV2State.current);
	}

	ngOnInit(): void {
		this.panelSize = this._store.selectSnapshot(PreferencesState.panelSize(ProjectV2ExploreComponent.PANEL_KEY));
	}

	panelStartResize(): void {
		this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: true }));
	}

	panelEndResize(size: string): void {
		this._store.dispatch(new actionPreferences.SavePanelSize({ panelKey: ProjectV2ExploreComponent.PANEL_KEY, size: size }));
		this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: false }));
	}
}