import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy } from '@angular/core';
import { PipelineStatus } from 'app/model/pipeline.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { GraphNode } from '../workflowv3.model';

@Component({
    selector: 'app-workflowv3-job-node',
    templateUrl: './workflowv3-job-node.html',
    styleUrls: ['./workflowv3-job-node.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowV3JobNodeComponent implements OnDestroy {
    @Input() node: GraphNode;
    @Input() mouseCallback: (type: string, node: GraphNode) => void;

    highlight = false;
    selected = false;
    pipelineStatusEnum = PipelineStatus;

    constructor(
        private _cd: ChangeDetectorRef
    ) {
        this.setHighlight.bind(this);
        this.setSelect.bind(this);
    }
    getNodes() { return [this.node]; }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    onMouseEnter(): void {
        if (this.mouseCallback) {
            this.mouseCallback('enter', this.node);
        }
    }

    onMouseOut(): void {
        if (this.mouseCallback) {
            this.mouseCallback('out', this.node);
        }
    }

    onMouseClick(): void {
        if (this.mouseCallback) {
            this.mouseCallback('click', this.node);
        }
    }

    setHighlight(active: boolean): void {
        this.highlight = active;
        this._cd.markForCheck();
    }

    setSelect(active: boolean): void {
        this.selected = active;
        this._cd.markForCheck();
    }
}
