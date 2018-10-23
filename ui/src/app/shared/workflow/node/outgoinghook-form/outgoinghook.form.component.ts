import {Component, EventEmitter, Input, OnInit, Output} from '@angular/core';
import {cloneDeep} from 'lodash';
import {Subscription} from 'rxjs';
import {finalize} from 'rxjs/operators';
import {IdName, Project} from '../../../../model/project.model';
import {WorkflowHookModel} from '../../../../model/workflow.hook.model';
import {
    WNode,
    WNodeHook, WNodeOutgoingHook, WNodeType, Workflow,
} from '../../../../model/workflow.model';
import {HookService} from '../../../../service/hook/hook.service';
import {WorkflowStore} from '../../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflow-node-outgoinghook-form',
    templateUrl: './outgoinghook.form.html',
    styleUrls: ['./outgoinghook.form.scss']
})
@AutoUnsubscribe()
export class WorkflowNodeOutGoingHookFormComponent implements OnInit {

    @Input() project: Project;
    @Input() workflow: Workflow;

    _outgoingHook: WNode;
    @Input('hook')
    set hook(data: WNode) {
        this._outgoingHook = data;
    }
    get hook() {
        return this._outgoingHook;
    }

    @Output() outgoinghookEvent = new EventEmitter<WNode>();

    codeMirrorConfig: {};
    loadingModels = false;
    outgoingHookModels: Array<WorkflowHookModel>;
    selectedOutgoingHookModel: WorkflowHookModel;
    displayConfig: boolean;
    availableWorkflows: Array<IdName>;
    loadingHooks: boolean;
    availableHooks: Array<WNodeHook>;
    wSub: Subscription;
    invalidJSON = false;

    constructor(private _hookService: HookService, private _workflowStore: WorkflowStore) {
    }

    ngOnInit(): void {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true,
        };

        this.loadingModels = true;
        this._hookService.getOutgoingHookModel().pipe(finalize(() => this.loadingModels = false))
            .subscribe(ms => {
            this.outgoingHookModels = ms;
            if (this.hook) {
                // select model
                if (this.hook.outgoing_hook.hook_model_id) {
                    this.selectedOutgoingHookModel = this.outgoingHookModels
                        .find(m => m.id === this.hook.outgoing_hook.hook_model_id);
                }
                this.displayConfig = Object.keys(this._outgoingHook.outgoing_hook.config).length !== 0;
                if (this.selectedOutgoingHookModel.name === 'Workflow') {
                    this.updateWorkflow();
                }
                this.availableWorkflows = this.project.workflow_names.filter(idName => idName.name !== this.workflow.name);
            }
        });
    }

    updateOutgoingHook(): void {
        if (!this._outgoingHook) {
            this._outgoingHook = new WNode();
            this._outgoingHook.type = WNodeType.OUTGOINGHOOK;
            this._outgoingHook.outgoing_hook = new WNodeOutgoingHook();
        }
        this._outgoingHook.outgoing_hook.hook_model_id = this.selectedOutgoingHookModel.id;
        this._outgoingHook.outgoing_hook.config = cloneDeep(this.selectedOutgoingHookModel.default_config);
        this.displayConfig = Object.keys(this._outgoingHook.outgoing_hook.config).length !== 0;

        // Specific behavior for the 'workflow' hooks
        if (this.selectedOutgoingHookModel.name === 'Workflow') {
            // Current limitation: trigger only workflow in the same project
            this._outgoingHook.outgoing_hook.config['target_project'].value = this.project.key;
            // Load the workflow for the current project, but exclude the current workflow
            this.availableWorkflows = this.project.workflow_names.filter(idName => idName.name !== this.workflow.name);
        }
    }

    updateWorkflow(): void {
        if (this.hook && this.hook.outgoing_hook.config && this.hook.outgoing_hook.config['target_project']
            &&  this.hook.outgoing_hook.config['target_workflow']) {
            this.loadingHooks = true;
            this.wSub = this._workflowStore.getWorkflows(this.hook.outgoing_hook.config['target_project'].value,
                this.hook.outgoing_hook.config['target_workflow'].value)
                .subscribe(
                    data => {
                        let key = this.project.key + '-' + this.hook.outgoing_hook.config['target_workflow'].value;
                        let wf = data.get(key);
                        if (wf) {
                            let allHooks = Workflow.getAllHooks(wf);
                            if (allHooks) {
                                this.availableHooks = allHooks.filter(h => wf.hook_models[h.hook_model_id].name === 'Workflow');
                            } else {
                                this.availableHooks = [];
                            }
                            if (this.hook || this.hook.outgoing_hook.config['target_hook']) {
                                if (this.availableHooks
                                    .findIndex( h =>
                                        h.uuid === this.hook.outgoing_hook.config['target_hook'].value) === -1) {
                                    this._outgoingHook.outgoing_hook.config['target_hook'].value = undefined;
                                }
                            }
                            this.loadingHooks = false;
                        }
                    }, () => {this.loadingHooks = false}
                );
        } else {
            this.availableHooks = null;
            if (this.wSub) {
                this.wSub.unsubscribe();
            }
        }

    }

    changeCodeMirror(code: string) {
        if (typeof code === 'string') {
            this.invalidJSON = false;
            try {
                JSON.parse(code);
            } catch (e) {
                this.invalidJSON = true;
            }
        }
    }
}
