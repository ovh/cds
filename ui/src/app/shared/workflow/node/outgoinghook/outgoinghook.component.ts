import { AfterViewInit, Component, ElementRef, Input } from '@angular/core';
import { Subscription } from 'rxjs';
import { Project } from '../../../../model/project.model';
import {
    Workflow,
    WorkflowNodeHookConfigValue,
    WorkflowNodeOutgoingHook
} from '../../../../model/workflow.model';
import { WorkflowEventStore } from '../../../../service/services.module';

@Component({
    selector: 'app-workflow-node-outgoinghook',
    templateUrl: './outgoinghook.html',
    styleUrls: ['./outgoinghook.scss']
})
export class WorkflowNodeOutgoingHookComponent implements AfterViewInit {

    _hook: WorkflowNodeOutgoingHook;

    @Input('hook')
    set hook(data: WorkflowNodeOutgoingHook) {
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

    icon: string;
    loading = false;
    isSelected = false;
    subSelect: Subscription;

    constructor(private elementRef: ElementRef, private _workflowEventStore: WorkflowEventStore) {
        this.subSelect = this._workflowEventStore.selectedOutgoingHook().subscribe(h => {
            if (this.hook && h) {
                this.isSelected = h.id === this.hook.id;
                return;
            }
            this.isSelected = false;
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
        this._workflowEventStore.setSelectedOutgoingHook(this.hook);
    }
}
