import {AfterViewInit, Component, ElementRef, Input, ViewChild} from '@angular/core';
import {Workflow, WorkflowNode, WorkflowNodeHook, WorkflowNodeHookConfigValue} from '../../../../model/workflow.model';
import {ToastService} from '../../../toast/ToastService';
import {TranslateService} from '@ngx-translate/core';
import {WorkflowNodeHookFormComponent} from './form/node.hook.component';
import {Project} from '../../../../model/project.model';
import {HookEvent} from './hook.event';
import {cloneDeep} from 'lodash';
import {WorkflowStore} from '../../../../service/workflow/workflow.store';
import {finalize} from 'rxjs/operators';

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

    @ViewChild('editHook')
    editHook: WorkflowNodeHookFormComponent;
    icon: string;
    loading = false;

    constructor(private elementRef: ElementRef, private _workflowStore: WorkflowStore, private _toast: ToastService,
                private _translate: TranslateService) {
    }

    ngAfterViewInit() {
        this.elementRef.nativeElement.style.position = 'fixed';
        this.elementRef.nativeElement.style.top = '5px';
    }

    openEditHookModal(): void {
        if (this.editHook && !this.readonly)  {
            this.editHook.show();
        }
    }

    updateHook(h: HookEvent): void {
        let workflowToUpdate = cloneDeep(this.workflow);
        this.loading = true;
        if (h.type === 'delete') {
            Workflow.removeHook(workflowToUpdate, h.hook);
        } else {
            Workflow.updateHook(workflowToUpdate, h.hook);
        }

        this._workflowStore.updateWorkflow(workflowToUpdate.project_key, workflowToUpdate)
            .pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this.editHook.modal.approve(true);
                this._toast.success('', this._translate.instant('workflow_updated'));
            });
    }
}
