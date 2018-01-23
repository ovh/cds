import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {
    Workflow, WorkflowNode, WorkflowNodeHook, WorkflowTriggerConditionCache
} from '../../../../../model/workflow.model';
import {HookService} from '../../../../../service/hook/hook.service';
import {WorkflowHookModel} from '../../../../../model/workflow.hook.model';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {cloneDeep} from 'lodash';
import {Project} from '../../../../../model/project.model';
import {WorkflowStore} from '../../../../../service/workflow/workflow.store';
import {HookEvent} from '../hook.event';
import {first, finalize} from 'rxjs/operators';
import {Observable} from 'rxjs/Observable';

@Component({
    selector: 'app-workflow-node-hook-form',
    templateUrl: './hook.form.html',
    styleUrls: ['./hook.form.scss']
})
export class WorkflowNodeHookFormComponent {

    _hook: WorkflowNodeHook = new WorkflowNodeHook();

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() node: WorkflowNode;
    @Input() loading: boolean;
    @Input('hook')
    set hook(data: WorkflowNodeHook) {
        if (data) {
            this._hook = data;
            if (this.hooksModel) {
                this.selectedHookModel = this.hooksModel.find(hm => hm.id === this._hook.model.id);
            }
            this.displayConfig = Object.keys(this._hook.config).length !== 0;
        }
    }
    get hook() {
        return this._hook;
    }
    @Input('hooksModel')
    set hooksModel(data: Array<WorkflowHookModel>) {
      this._hooksModel = data;
      if (this.hook && this.hook.model) {
          this.selectedHookModel = data.find(hm => hm.id === this._hook.model.id);
      }
    }
    get hooksModel() {
      return this._hooksModel;
    }

    selectedHookModel: WorkflowHookModel;
    _hooksModel: Array<WorkflowHookModel>;
    displayConfig = false;

    constructor(private _hookService: HookService, private _workflowStore: WorkflowStore) {
    }

    updateHook(): void {
        this.hook.model = this.selectedHookModel;
        this.hook.config = cloneDeep(this.selectedHookModel.default_config);
        this.displayConfig = Object.keys(this.hook.config).length !== 0;
    }
}
