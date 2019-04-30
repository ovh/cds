import {Component, Input, OnInit} from '@angular/core';
import {cloneDeep} from 'lodash';
import {finalize, first} from 'rxjs/operators';
import {WNode, WNodeHook, Workflow, WorkflowNodeHookConfigValue} from 'app/model/workflow.model';
import {Project} from 'app/model/project.model';
import {WorkflowHookModel} from 'app/model/workflow.hook.model';
import {ProjectIntegration} from 'app/model/integration.model';
import {HookService} from 'app/service/hook/hook.service';
import {UpdateWorkflow} from 'app/store/workflows.action';
import {Store} from '@ngxs/store';
import {ToastService} from 'app/shared/toast/ToastService';
import {TranslateService} from '@ngx-translate/core';

@Component({
    selector: 'app-workflow-node-hook-form',
    templateUrl: './hook.form.html',
    styleUrls: ['./hook.form.scss']
})
export class WorkflowNodeHookFormComponent implements OnInit {

    _hook: WNodeHook = new WNodeHook();
    canDelete = false;

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() node: WNode;
    @Input('hook')
    set hook(data: WNodeHook) {
        if (data) {
            this.canDelete = true;
            this._hook = cloneDeep<WNodeHook>(data);
            if (this.hooksModel) {
                this.selectedHookModel = this.hooksModel.find(hm => hm.id === this.hook.hook_model_id);
                this._hook.model = this.selectedHookModel;
            }
            this.displayConfig = Object.keys(this._hook.config).length !== 0;
        }
    }
    get hook() {
        return this._hook;
    }

    // Enable form button to update hook
    @Input() formMode = false;

    hooksModel: Array<WorkflowHookModel>;
    selectedHookModel: WorkflowHookModel;
    readonly = false;
    loadingModels = true;
    displayConfig = false;
    invalidJSON = false;
    updateMode = false;
    codeMirrorConfig: any;
    selectedIntegration: ProjectIntegration;
    availableIntegrations: Array<ProjectIntegration>;
    loading = false;

    constructor(private _hookService: HookService, private _store: Store, private _toast: ToastService,
                private _translate: TranslateService) { }

    updateHook(): void {
        this.loading = true;
        let clonedWorkflow = cloneDeep(this.workflow);
        let n = Workflow.getNodeByID(this.node.id, clonedWorkflow);
        let h = WNode.getHookByRef(n, this.hook.ref);
        if (h) {
            h.config = cloneDeep(this.hook.config);
            this._store.dispatch(new UpdateWorkflow({
                projectKey: this.workflow.project_key,
                workflowName: this.workflow.name,
                changes: clonedWorkflow
            })).pipe(finalize(() => this.loading = false))
                .subscribe(() => {
                    this._toast.success('', this._translate.instant('workflow_updated'));
                });
        }
    }

    updateHookModel(): void {
        this.hook.model = this.selectedHookModel;
        this.hook.config = cloneDeep(this.selectedHookModel.default_config);
        this.hook.hook_model_id = this.selectedHookModel.id;
        this.initConfig();

        this.displayConfig = Object.keys(this.hook.config).length !== 0;
    }

    initConfig(): void {
        Object.getOwnPropertyNames(this.hook.config).forEach(k => {
            if ((<WorkflowNodeHookConfigValue>this.hook.config[k]).type === 'multiple') {

                // init temp array for checkbox
                (<WorkflowNodeHookConfigValue>this.hook.config[k]).temp = {};
                for (let event of this.hook.config[k].value.split(';')) {
                    (<WorkflowNodeHookConfigValue>this.hook.config[k]).temp[event] = true;
                }

                // init ref list from model
                (<WorkflowNodeHookConfigValue>this.hook.config[k]).multiple_choice_list =
                    (<WorkflowNodeHookConfigValue>this.hook.model.default_config[k]).multiple_choice_list;
            }

        });
    }

    updateHookMultiChoice(k: string): void {
        let finalValue = Object.getOwnPropertyNames((<WorkflowNodeHookConfigValue>this.hook.config[k]).temp).filter( choice => {
            return (<WorkflowNodeHookConfigValue>this.hook.config[k]).temp[choice];
        });
        (<WorkflowNodeHookConfigValue>this.hook.config[k]).value = finalValue.join(';');
    }

    updateIntegration(): void {
        Object.keys(this.hook.config).forEach( k => {
            if (k === 'integration') {
                this.hook.config[k].value = this.selectedIntegration.name;
            } else {
                if (this.selectedIntegration.config[k]) {
                    this.hook.config[k] = cloneDeep(this.selectedIntegration.config[k])
                }
            }
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

    ngOnInit(): void {
        this.availableIntegrations = this.project.integrations.filter(pf =>  pf.model.hook);
        if (this.hook && this.hook.config && this.hook.config['integration']) {
            this.selectedIntegration = this.project.integrations.find(pf => pf.name === this.hook.config['integration'].value);
        }
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true,
            readOnly: this.readonly,
        };
        this.loadingModels = true;
        if (!this.node && this.hook) {
            this.node = Workflow.getNodeByID(this.hook.node_id, this.workflow);
        }
        this._hookService.getHookModel(this.project, this.workflow, this.node).pipe(
            first(),
            finalize(() => this.loadingModels = false)
        ).subscribe(mds => {
            this.hooksModel = mds;
            if (this.hook && this.hook.hook_model_id) {
                this.selectedHookModel = this.hooksModel.find(hm => hm.id === this.hook.hook_model_id);
                this.hook.model = this.selectedHookModel;
                this.initConfig();
            }
        });
    }
}
