import {AfterViewInit, Component, ElementRef, Input} from '@angular/core';
import {Workflow, WorkflowNode, WorkflowNodeHook, WorkflowNodeHookConfigValue} from '../../../../model/workflow.model';
import {Project} from '../../../../model/project.model';
import {WorkflowEventStore} from '../../../../service/workflow/workflow.event.store';
import {Subscription} from 'rxjs/Subscription';

@Component({
    selector: 'app-workflow-node-hook',
    templateUrl: './hook.html',
    styleUrls: ['./hook.scss']
})
export class WorkflowNodeHookComponent implements AfterViewInit {

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
    @Input() project: Project;
    @Input() node: WorkflowNode;

    icon: string;
    loading = false;
    isSelected = false;
    subSelect: Subscription;

    constructor(private elementRef: ElementRef, private _workflowEventStore: WorkflowEventStore) {

        this.subSelect = this._workflowEventStore.selectedHook().subscribe(h => {
            if (this.hook && h) {
                this.isSelected = h.id === this.hook.id;
                return;
            }
            this.isSelected = false;
            return;

        });
    }

    ngAfterViewInit() {
        this.elementRef.nativeElement.style.position = 'fixed';
        this.elementRef.nativeElement.style.top = '5px';
    }

    openEditHookSidebar(): void {
        if (this.workflow.previewMode) {
          return;
        }
        this._workflowEventStore.setSelectedHook(this.hook);
    }
}
