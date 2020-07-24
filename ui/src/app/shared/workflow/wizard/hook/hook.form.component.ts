import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Select, Store } from '@ngxs/store';
import { ProjectIntegration } from 'app/model/integration.model';
import { Project } from 'app/model/project.model';
import { WorkflowHookModel } from 'app/model/workflow.hook.model';
import { WNode, WNodeHook, Workflow, WorkflowNodeHookConfigValue } from 'app/model/workflow.model';
import { HookService } from 'app/service/hook/hook.service';
import { ThemeStore } from 'app/service/theme/theme.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { ProjectState } from 'app/store/project.state';
import { UpdateWorkflow } from 'app/store/workflow.action';
import { WorkflowState } from 'app/store/workflow.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Observable } from 'rxjs';
import { finalize, first } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';

@Component({
    selector: 'app-workflow-node-hook-form',
    templateUrl: './hook.form.html',
    styleUrls: ['./hook.form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowNodeHookFormComponent implements OnInit, OnDestroy {

    @Input() workflow: Workflow;

    // Enable form button to update hook
    @Input() mode = 'create'; // create  update ro

    @ViewChild('textareaCodeMirror') codemirror: any;

    @Select(WorkflowState.getSelectedNode()) node$: Observable<WNode>;
    node: WNode;
    nodeSub: Subscription;

    @Select(WorkflowState.getSelectedHook()) hook$: Observable<WNodeHook>;
    hook: WNodeHook;
    hookSub: Subscription;


    project: Project;
    editMode: boolean;
    isRun: boolean;

    canDelete = false;
    hooksModel: Array<WorkflowHookModel>;
    selectedHookModel: WorkflowHookModel;
    loadingModels = true;
    displayConfig = false;
    invalidJSON = false;
    updateMode = false;
    codeMirrorConfig: any;
    selectedIntegration: ProjectIntegration;
    availableIntegrations: Array<ProjectIntegration>;
    loading = false;
    themeSubscription: Subscription;
    tempMultipleConfig = [];

    constructor(
        private _hookService: HookService,
        private _store: Store,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _theme: ThemeStore,
        private _cd: ChangeDetectorRef
    ) {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
        this.editMode = this._store.selectSnapshot(WorkflowState).editMode;
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.nodeSub = this.node$.subscribe(n => {
            this.node = n;
            this._cd.markForCheck();
        });
        this.hookSub = this.hook$.subscribe(h => {
            if (!h) {
                return;
            }
            this.hook = cloneDeep(h);
            this.canDelete = true;
            if (this.hooksModel) {
                this.selectedHookModel = this.hooksModel.find(hm => hm.id === this.hook.hook_model_id);
                this.hook.model = this.selectedHookModel;
            }
            this.displayConfig = Object.keys(this.hook.config).length !== 0;
            this._cd.markForCheck();
        });
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true,
            readOnly: this.mode === 'ro',
        };

        this.themeSubscription = this._theme.get().pipe(finalize(() => this._cd.markForCheck())).subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
        });

        this.availableIntegrations = this.project.integrations?.filter(pf => pf.model.hook);
        if (this.hook && this.hook.config && this.hook.config['integration']) {
            this.selectedIntegration = this.project.integrations.find(pf => pf.name === this.hook.config['integration'].value);
        }
        this.loadingModels = true;
        if (!this.node && this.hook) {
            this.node = Workflow.getNodeByID(this.hook.node_id, this.workflow);
        }
        if (!this._store.selectSnapshot(WorkflowState).workflowRun) {
            this._hookService.getHookModel(this.project, this.workflow, this.node).pipe(
                first(),
                finalize(() => {
                    this.loadingModels = false;
                    this._cd.markForCheck();
                })
            ).subscribe(mds => {
                this.hooksModel = mds;
                if (this.hook && this.hook.hook_model_id) {
                    this.selectedHookModel = this.hooksModel.find(hm => hm.id === this.hook.hook_model_id);
                    this.hook.model = this.selectedHookModel;
                    this.initConfig();
                }
            });
        } else {
            this.isRun = true;
        }
    }

    updateHook(): void {
        this.loading = true;
        let clonedWorkflow = cloneDeep(this.workflow);
        let n: WNode;
        if (this.editMode) {
            n = Workflow.getNodeByRef(this.node.ref, clonedWorkflow);
        } else {
            n = Workflow.getNodeByID(this.node.id, clonedWorkflow);
        }
        let h = WNode.getHookByRef(n, this.hook.ref);
        if (h) {
            h.config = cloneDeep(this.hook.config);
            this._store.dispatch(new UpdateWorkflow({
                projectKey: this.workflow.project_key,
                workflowName: this.workflow.name,
                changes: clonedWorkflow
            })).pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
                .subscribe(() => {
                    if (this.editMode) {
                        this._toast.info('', this._translate.instant('workflow_ascode_updated'));
                    } else {
                        this._toast.success('', this._translate.instant('workflow_updated'));
                    }
                });
        }
    }

    updateHookModel(): void {
        if (!this.hook) {
            this.hook = new WNodeHook();
        }
        this.hook.model = this.selectedHookModel;
        this.hook.config = cloneDeep(this.selectedHookModel.default_config);
        this.hook.hook_model_id = this.selectedHookModel.id;
        this.initConfig();

        this.displayConfig = Object.keys(this.hook.config).length !== 0;
    }

    initConfig(): void {
        Object.getOwnPropertyNames(this.hook.config).forEach(k => {
            if ((<WorkflowNodeHookConfigValue>this.hook.config[k]).type === 'multiple') {
                this.tempMultipleConfig = [];
                for (let event of this.hook.config[k].value.split(';')) {
                    this.tempMultipleConfig.push(event);
                }
                // init ref list from model
                (<WorkflowNodeHookConfigValue>this.hook.config[k]).multiple_choice_list =
                    (<WorkflowNodeHookConfigValue>this.hook.model.default_config[k]).multiple_choice_list;
            }

        });
    }

    updateConfigMultipleChoice(k: string): void {
        (<WorkflowNodeHookConfigValue>this.hook.config[k]).value = this.tempMultipleConfig.join(';');
    }

    updateIntegration(): void {
        Object.keys(this.hook.config).forEach(k => {
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
}
