import {AfterViewInit, Component, ElementRef, Input} from '@angular/core';
import {Subscription} from 'rxjs';
import {Workflow, WorkflowNodeFork} from '../../../model/workflow.model';
import {WorkflowEventStore} from '../../../service/workflow/workflow.event.store';
import {AutoUnsubscribe} from '../../decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflow-node-fork',
    templateUrl: './fork.html',
    styleUrls: ['./fork.scss']
})
@AutoUnsubscribe()
export class WorkflowNodeForkComponent implements AfterViewInit {

    @Input() workflow: Workflow;
    @Input() fork: WorkflowNodeFork;

    isSelected = false;

    subSelect: Subscription;

    constructor(private elementRef: ElementRef, private _workflowEventStore: WorkflowEventStore) {
        this.subSelect = this._workflowEventStore.selectedFork().subscribe(f => {
            if (this.fork && f) {
                this.isSelected = f.id === this.fork.id;
                return;
            }
            this.isSelected = false;
        });
    }

    ngAfterViewInit(): void {
        this.elementRef.nativeElement.style.position = 'fixed';
    }

    openEditForkSidebar(): void {
        if (this.workflow.previewMode) {
            return;
        }
        this._workflowEventStore.setSelectedFork(this.fork);
    }
}
