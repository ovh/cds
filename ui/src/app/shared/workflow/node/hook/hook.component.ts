import {AfterViewInit, Component, ElementRef, Input, OnInit} from '@angular/core';
import { WorkflowNodeRun, WorkflowRun } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import {Subscription} from 'rxjs/Subscription';
import {Project} from '../../../../model/project.model';
import {Workflow, WorkflowNode, WorkflowNodeHook, WorkflowNodeHookConfigValue} from '../../../../model/workflow.model';
import {WorkflowEventStore} from '../../../../service/workflow/workflow.event.store';

@Component({
    selector: 'app-workflow-node-hook',
    templateUrl: './hook.html',
    styleUrls: ['./hook.scss']
})
@AutoUnsubscribe()
export class WorkflowNodeHookComponent implements AfterViewInit, OnInit {
    _hook: WorkflowNodeHook;
    @Input('hook')
    set hook(data: WorkflowNodeHook) {
        if (data) {
            this._hook = data;
            if (this._hook.config['hookIcon']) {
                this.icon = (<WorkflowNodeHookConfigValue>this._hook.config['hookIcon']).value.toLowerCase();
            } else {
                this.icon = this._hook.model.icon.toLowerCase();
            }
        }
    }
    get hook() {
      return this._hook;
    }
    @Input() readonly = false;
    @Input() workflow: Workflow;
    @Input() workflowRun: WorkflowRun;
    @Input() project: Project;
    @Input() node: WorkflowNode;

    icon: string;
    loading = false;
    isSelected = false;
    subSelect: Subscription;
    subRun: Subscription;
    nodeRun: WorkflowNodeRun;

    constructor(private elementRef: ElementRef, private _workflowEventStore: WorkflowEventStore) {}

    ngAfterViewInit() {
        this.elementRef.nativeElement.style.position = 'fixed';
        this.elementRef.nativeElement.style.top = '5px';
    }

    ngOnInit(): void {
        this.subSelect = this._workflowEventStore.selectedHook().subscribe(h => {
            if (this.hook && h) {
                this.isSelected = h.id === this.hook.id;
                return;
            }
            this.isSelected = false;
        });

        // Get workflow run
        this.subRun = this._workflowEventStore.selectedRun().subscribe(wr => {
            this.workflowRun = wr;
            if (wr && wr.nodes && this.node && wr.nodes[this.node.id] && wr.nodes[this.node.id].length > 0) {
                this.nodeRun = this.workflowRun.nodes[this.node.id][0];
            } else {
                this.nodeRun = null;
            }
        });
    }

    openEditHookSidebar(): void {
        if (this.workflow.previewMode) {
          return;
        }
        this._workflowEventStore.setSelectedHook(this.hook);
    }
}
