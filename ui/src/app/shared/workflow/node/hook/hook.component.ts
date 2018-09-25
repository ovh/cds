import {AfterViewInit, Component, ElementRef, Input, OnInit} from '@angular/core';
import {Subscription} from 'rxjs/Subscription';
import {Project} from '../../../../model/project.model';
import {WNode, WNodeHook, Workflow, WorkflowNodeHookConfigValue} from '../../../../model/workflow.model';
import {WorkflowEventStore} from '../../../../service/workflow/workflow.event.store';

@Component({
    selector: 'app-workflow-node-hook',
    templateUrl: './hook.html',
    styleUrls: ['./hook.scss']
})
export class WorkflowNodeHookComponent implements OnInit, AfterViewInit {

    _hook: WNodeHook;
    @Input('hook')
    set hook(data: WNodeHook) {
        if (data) {
            this._hook = data;
        }
    }
    get hook() {
      return this._hook;
    }
    @Input() readonly = false;
    @Input() workflow: Workflow;
    @Input() project: Project;
    @Input() node: WNode;

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
        });
    }

    ngOnInit(): void {
        if (this._hook) {
            if (this._hook.config['hookIcon']) {
                this.icon = (<WorkflowNodeHookConfigValue>this._hook.config['hookIcon']).value.toLowerCase();
            } else {
                this.icon = this.workflow.hook_models[this.hook.hook_model_id].icon.toLowerCase();
            }
        }
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
