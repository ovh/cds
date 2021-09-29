import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { Project } from 'app/model/project.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { GraphDirection } from '../graph/workflowv3-graph.lib';
import { WorkflowV3StagesGraphComponent } from '../graph/workflowv3-stages-graph.component';
import { WorkflowV3 } from '../workflowv3.model';

@Component({
    selector: 'app-workflowv3-show',
    templateUrl: './workflowv3-show.html',
    styleUrls: ['./workflowv3-show.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowV3ShowComponent implements OnInit, OnDestroy {
    @ViewChild('graph') graph: WorkflowV3StagesGraphComponent;

    data: WorkflowV3;
    direction: GraphDirection = GraphDirection.VERTICAL;
    project: Project;
    resizing = false;

    constructor(
        private _cd: ChangeDetectorRef
    ) { }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void { }

    workflowEdit(data: WorkflowV3) {
        this.data = data;
        this._cd.markForCheck();
    }

    panelStartResize(): void {
        this.resizing = true;
        this._cd.markForCheck();
    }

    panelEndResize(): void {
        this.resizing = false;
        this._cd.markForCheck();
        if (this.graph) {
            this.graph.resize();
        }
    }
}
