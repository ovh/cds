import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { Store } from '@ngxs/store';
import { Project } from 'app/model/project.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { PreferencesState } from 'app/store/preferences.state';
import { GraphDirection } from '../graph/workflowv3-graph.lib';
import { WorkflowV3StagesGraphComponent } from '../graph/workflowv3-stages-graph.component';
import { WorkflowV3 } from '../workflowv3.model';
import * as actionPreferences from 'app/store/preferences.action';
import { Subscription } from 'rxjs';

@Component({
    selector: 'app-workflowv3-show',
    templateUrl: './workflowv3-show.html',
    styleUrls: ['./workflowv3-show.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowV3ShowComponent implements OnInit, OnDestroy {
    static PANEL_KEY = 'workflow-v3-show';

    @ViewChild('graph') graph: WorkflowV3StagesGraphComponent;

    data: WorkflowV3;
    direction: GraphDirection = GraphDirection.VERTICAL;
    project: Project;
    resizing = false;
    panelSize: number;
    resizingSubscription: Subscription;

    constructor(
        private _cd: ChangeDetectorRef,
        private _store: Store
    ) { }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.panelSize = this._store.selectSnapshot(PreferencesState.panelSize(WorkflowV3ShowComponent.PANEL_KEY));

        this.resizingSubscription = this._store.select(PreferencesState.resizing).subscribe(resizing => {
            this.resizing = resizing;
            this._cd.markForCheck();
        });

        this._cd.markForCheck();
    }

    workflowEdit(data: WorkflowV3) {
        this.data = data;
        this._cd.markForCheck();
    }

    panelStartResize(): void {
        this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: true }));
    }

    panelEndResize(size: number): void {
        this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: false }));
        this._cd.detectChanges(); // force rendering to compute graph container size
        if (this.graph) {
            this.graph.resize();
        }
        this._store.dispatch(new actionPreferences.SavePanelSize({ panelKey: WorkflowV3ShowComponent.PANEL_KEY, size: size }));
    }
}
