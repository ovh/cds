import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import {PermissionValue} from 'app/model/permission.model';
import {ToastService} from 'app/shared/toast/ToastService';
import {FetchWorkflow, UpdateWorkflow} from 'app/store/workflows.action';
import { WorkflowsState } from 'app/store/workflows.state';
import { cloneDeep } from 'lodash';
import { Subscription } from 'rxjs';
import { finalize, flatMap } from 'rxjs/operators';
import { IdName, Project } from '../../../../model/project.model';
import { WorkflowHookModel } from '../../../../model/workflow.hook.model';
import { WNode, WNodeHook, WNodeOutgoingHook, WNodeType, Workflow } from '../../../../model/workflow.model';
import { HookService } from '../../../../service/hook/hook.service';
import { AutoUnsubscribe } from '../../../decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflow-node-outgoinghook',
    templateUrl: './wizard.outgoinghook.html',
    styleUrls: ['./wizard.outgoinghook.scss']
})
@AutoUnsubscribe()
export class WorkflowWizardOutgoingHookComponent implements OnInit {

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() updateMode = false;

    _outgoingHook: WNode;
    @Input('hook')
    set hook(data: WNode) {
        this._outgoingHook = data;
    }
    get hook() {
        return this._outgoingHook;
    }

    @Output() outgoinghookEvent = new EventEmitter<WNode>();

    permissionEnum = PermissionValue;
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
    outgoing_default_payload: {};
    loading = false;

    constructor(
        private store: Store, private _translate: TranslateService, private _toast: ToastService,
        private _hookService: HookService
    ) {
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
                        this.updateWorkflowData();
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

    updateWorkflowData(): void {
        if (this.hook && this.hook.outgoing_hook.config && this.hook.outgoing_hook.config['target_project']
            && this.hook.outgoing_hook.config['target_workflow']) {
            this.loadingHooks = true;
            this.store.dispatch(new FetchWorkflow({
                projectKey: this.hook.outgoing_hook.config['target_project'].value,
                workflowName: this.hook.outgoing_hook.config['target_workflow'].value
            })).pipe(
                flatMap(() => {
                    return this.store.selectOnce(WorkflowsState.selectWorkflow(
                        this.hook.outgoing_hook.config['target_project'].value,
                        this.hook.outgoing_hook.config['target_workflow'].value)
                    );
                }),
                finalize(() => this.loadingHooks = false)
            ).subscribe((wf: Workflow) => {
                this.outgoing_default_payload = wf.workflow_data.node.context.default_payload;
                let allHooks = Workflow.getAllHooks(wf);
                if (allHooks) {
                    this.availableHooks = allHooks.filter(h => wf.hook_models[h.hook_model_id].name === 'Workflow');
                } else {
                    this.availableHooks = [];
                }
                if (this.hook || this.hook.outgoing_hook.config['target_hook']) {
                    if (this.availableHooks
                        .findIndex(h =>
                            h.uuid === this.hook.outgoing_hook.config['target_hook'].value) === -1) {
                        this._outgoingHook.outgoing_hook.config['target_hook'].value = undefined;
                    }
                }
            });
        } else {
            this.availableHooks = null;
            if (this.wSub) {
                this.wSub.unsubscribe();
            }
        }
    }

    updateWorkflowOutgoingHook(): void {
        if (this.hook.outgoing_hook.config['target_hook']) {
            this.hook.outgoing_hook.config['payload'].value = JSON.stringify(this.outgoing_default_payload, undefined, 4);
        }
    }

    updateWorkflow(): void {
        this.loading = true;
        let clonedWorkflow = cloneDeep(this.workflow);
        let n = Workflow.getNodeByID(this.hook.id, clonedWorkflow);

        n.outgoing_hook = this.hook.outgoing_hook;
        this.store.dispatch(new UpdateWorkflow({
            projectKey: this.workflow.project_key,
            workflowName: this.workflow.name,
            changes: clonedWorkflow
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_updated'));
            });
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
