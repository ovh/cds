import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    Input,
    OnDestroy,
    OnInit,
    ViewChild
} from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { EventService } from 'app/event.service';
import { Project } from 'app/model/project.model';
import { RunToKeep } from 'app/model/purge.model';
import { Workflow, WorkflowProjectIntegration } from 'app/model/workflow.model';
import { FeatureNames } from 'app/service/feature/feature.service';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Column, ColumnType } from 'app/shared/table/data-table.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { FeatureState } from 'app/store/feature.state';
import {
    CleanRetentionDryRun,
    DeleteIntegrationWorkflow,
    DeleteWorkflowIcon,
    UpdateIntegrationsWorkflow,
    UpdateWorkflow,
    UpdateWorkflowIcon
} from 'app/store/workflow.action';
import { WorkflowState } from 'app/store/workflow.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { CodemirrorComponent } from 'ng2-codemirror-typescript/Codemirror';
import { DragulaService } from 'ng2-dragula';
import { forkJoin, Subscription } from 'rxjs';
import { finalize, first } from 'rxjs/operators';
import { ProjectIntegration } from 'app/model/integration.model';
import { APIConfig } from 'app/model/config.service';
import { WorkflowDeleteModalComponent } from './delete-modal/delete-modal.component';
import { NzModalService } from 'ng-zorro-antd/modal';
import { ConfigState } from 'app/store/config.state';
import { PreferencesState } from 'app/store/preferences.state';

declare let CodeMirror: any;

@Component({
    selector: 'app-workflow-admin',
    templateUrl: 'workflow.admin.component.html',
    styleUrls: ['./workflow.admin.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowAdminComponent implements OnInit, OnDestroy, AfterViewInit {

    _project: Project;
    @Input()
    set project(project: Project) {
        this._project = project;
        if (project.integrations) {
            this.filteredIntegrations = cloneDeep(project.integrations.filter(p => p.model.artifact_manager));
        }
    }

    get project(): Project {
        return this._project;
    }

    _workflow: Workflow;
    @Input()
    set workflow(data: Workflow) {
        if (data) {
            this._workflow = cloneDeep(data);
            this.nbEventIntegrations = this._workflow?.integrations?.filter(i => i.project_integration.model.event).length;
        }
    }

    get workflow() {
        return this._workflow;
    }

    @Input() editMode: boolean;

    oldName: string;

    runnumber: number;
    originalRunNumber: number;

    allTags = new Array<string>();
    existingTags = new Array<string>();
    selectedTags = new Array<string>();
    existingTagsPurge = new Array<string>();
    selectedTagsPurge = new Array<string>();
    iconUpdated = false;
    tagsToAdd = new Array<string>();
    tagsToAddPurge = new Array<string>();
    retentionRunsPolicyEnabled = false;
    maxRunsEnabled = false;
    codeMirrorConfig: any;

    filteredIntegrations: Array<ProjectIntegration>;
    nbEventIntegrations: number;
    selectedIntegration: ProjectIntegration;

    apiConfig: APIConfig;

    @ViewChild('codemirrorRetentionPolicy') codemirror: CodemirrorComponent;
    themeSubscription: Subscription;

    // Dry run datas
    dryRunsSubs: Subscription;
    dryRunsStatusSubs: Subscription;
    dryRunProgressSub: Subscription;
    dryRunWarningsSub: Subscription;
    dryRunColumns = [];
    dryRunDatas: Array<RunToKeep>;
    dryRunMaxDatas: number;
    dryRunStatus: string;
    dryRunWarnings: string[];
    dryRunAnalyzedRuns: number;
    availableVariables: Array<string>;
    availableStringVariables: string;
    _keyUpListener: any;

    dryRunModalVisible: boolean = false;

    loading = false;
    fileTooLarge = false;
    dragulaSubscription: Subscription;
    configSubscription: Subscription;

    constructor(
        private _store: Store,
        public _translate: TranslateService,
        private _toast: ToastService,
        private _router: Router,
        private _workflowRunService: WorkflowRunService,
        private _workflowService: WorkflowService,
        private _cd: ChangeDetectorRef,
        private _dragularService: DragulaService,
        private _nzModalService: NzModalService,
        private _eventService: EventService,
    ) {
        this._dragularService.createGroup('bag-tag', {
            accepts(el, target, source, sibling) {
                return sibling !== null;
            }
        });

        this.dragulaSubscription = this._dragularService.drop('bag-tag').subscribe(({ }) => {
            setTimeout(() => {
                this.updateTagMetadata();
            });
        });
        this.dryRunColumns = [
            <Column<RunToKeep>>{
                name: 'run_number',
                class: 'two',
                selector: (r: RunToKeep) => r.num
            },
            <Column<RunToKeep>>{
                type: ColumnType.TEXT,
                name: 'status',
                class: 'two',
                selector: (r: RunToKeep) => r.status
            }
        ];
    }

    ngOnDestroy(): void {
        this._dragularService.destroy('bag-tag');
        this._eventService.unsubscribeWorkflowRetention();
    }

    ngOnInit(): void {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'text/x-lua',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true,
            gutters: ['CodeMirror-lint-markers']
        };

        this.themeSubscription = this._store.select(PreferencesState.theme).subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
                this._cd.markForCheck();
            }
        });

        this.configSubscription = this._store.select(ConfigState.api).subscribe(c => {
            this.apiConfig = c;
            this._cd.markForCheck();
        });

        if (!this._workflow.metadata) {
            this._workflow.metadata = new Map<string, string>();
        }
        if (this._workflow.metadata['default_tags']) {
            this.selectedTags = this._workflow.metadata['default_tags'].split(',');
        }
        if (this._workflow.purge_tags && this._workflow.purge_tags.length) {
            this.selectedTagsPurge = this._workflow.purge_tags;
        }

        if (!this.project.permissions.writable) {
            this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'applications' } });
        }
        this.oldName = this.workflow.name;

        this._workflowRunService.getTags(this.project.key, this._workflow.name)
            .pipe(finalize(() => this._cd.markForCheck()))
            .subscribe(tags => {
                if (tags) {
                    this.allTags = Object.keys(tags);
                    this.initExistingtags();
                }
            });
        this._workflowRunService.getRunNumber(this.project.key, this.workflow)
            .pipe(first(), finalize(() => this._cd.markForCheck())).subscribe(n => {
                this.originalRunNumber = n.num;
                this.runnumber = n.num;
            });

        this.initDryRunSubscription();

        let featRetentionRunsPolicyResult = this._store.selectSnapshot(FeatureState.featureProject(FeatureNames.WorkflowRetentionPolicy,
            JSON.stringify({ project_key: this.project.key })));
        this.retentionRunsPolicyEnabled = featRetentionRunsPolicyResult?.enabled;
        let featMaxRunsResult = this._store.selectSnapshot(FeatureState.featureProject(FeatureNames.WorkflowRetentionMaxRuns,
            JSON.stringify({ project_key: this.project.key })));
        this.maxRunsEnabled = featMaxRunsResult?.enabled;

        this._cd.markForCheck();
    }

    changeCodeMirror(codemirror) {
        if (!this._keyUpListener) {
            this._keyUpListener = codemirror.instance.on('keyup', (cm, event) => {
                if (!cm.state.completionActive && (event.keyCode > 46 || event.keyCode === 32)) {
                    CodeMirror.showHint(cm, CodeMirror.hint.textplain, {
                        completeSingle: true,
                        closeCharacters: / /,
                        completionList: this.availableVariables,
                        specialChars: ''
                    });
                }
            });
        }
    }

    ngAfterViewInit() {
       this.codemirror.instance.refresh();
    }

    initExistingtags(): void {
        this.existingTags = [];
        this.existingTagsPurge = [];
        this.allTags.forEach(t => {
            if (this.selectedTags.indexOf(t) === -1) {
                this.existingTags.push(t);
            }
            if (this.selectedTagsPurge.indexOf(t) === -1) {
                this.existingTagsPurge.push(t);
            }
        });
    }

    initDryRunSubscription() {
        this._workflowService.retentionPolicySuggestion(this.workflow).subscribe(sg => {
            this.availableVariables = sg;
            this.availableStringVariables = sg.sort().join(', ');
            this._cd.markForCheck();
        });

        // Subscribe to dry run result update
        this.dryRunsSubs = this._store.select(WorkflowState.getRetentionDryRunResults()).subscribe(rs => {
            if (!this.dryRunDatas) {
                this.dryRunDatas = new Array<RunToKeep>();
            }
            if (this.dryRunDatas.length === rs.length) {
                return;
            }
            this.dryRunDatas = rs;
            this._cd.markForCheck();
        });
        // Subscribe to dry run result status
        this.dryRunsStatusSubs = this._store.select(WorkflowState.getRetentionStatus()).subscribe(s => {
            if (s === this.dryRunStatus) {
                return;
            }
            this.dryRunStatus = s;
            if (this.dryRunStatus === 'DONE') {
                this._eventService.unsubscribeWorkflowRetention();
            }
            this._cd.markForCheck();
        });
        this.dryRunProgressSub = this._store.select(WorkflowState.getRetentionProgress()).subscribe(nb => {
            if (nb === this.dryRunAnalyzedRuns) {
                return;
            }
            this.dryRunAnalyzedRuns = nb;
            this._cd.markForCheck();
        });
        this.dryRunWarningsSub = this._store.select(WorkflowState.getRetentionDryRunWarnings()).subscribe(ws => {
           this.dryRunWarnings = ws;
           this._cd.markForCheck();
        });

    }

    deleteIcon(): void {
        this.loading = true;
        this._store.dispatch(new DeleteWorkflowIcon({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => this._toast.success('', this._translate.instant('workflow_updated')));
    }

    updateIcon(): void {
        this.loading = true;
        this._store.dispatch(new UpdateWorkflowIcon({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            icon: this.workflow.icon
        })).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        }))
            .subscribe(() => {
                this.iconUpdated = false;
                this._toast.success('', this._translate.instant('workflow_updated'));
            });
    }

    updateTagMetadata(): void {
        if (this.tagsToAdd && this.tagsToAdd.length > 0) {
            if (!this.selectedTags) {
                this.selectedTags = [];
            }
            this.selectedTags.push(...this.tagsToAdd);
            this.initExistingtags();
        }

        this._workflow.metadata['default_tags'] = this.selectedTags.join(',');
        this.tagsToAdd = [];
        this._cd.markForCheck();
    }

    updateTagPurge(): void {
        if (this.tagsToAddPurge && this.tagsToAddPurge.length > 0) {
            if (!this.selectedTagsPurge) {
                this.selectedTagsPurge = [];
            }
            this.selectedTagsPurge.push(...this.tagsToAddPurge);
            this.initExistingtags();
        }

        this._workflow.purge_tags = this.selectedTagsPurge;
        this.tagsToAddPurge = [];
        this._cd.markForCheck();
    }

    removeFromSelectedTags(ind: number): void {
        this.selectedTags.splice(ind, 1);
        this.initExistingtags();
        this.updateTagMetadata();
    }

    removeFromSelectedTagsPurge(ind: number): void {
        this.selectedTagsPurge.splice(ind, 1);
        this.initExistingtags();
        this.updateTagPurge();
    }

    retentionPolicyDryRun(): void {
        this._store.dispatch(new CleanRetentionDryRun());
        this._eventService.subscribeToWorkflowPurgeDryRun(this.project.key, this.workflow.name);
        this.loading = true;
        this._workflowService.retentionPolicyDryRun(this.workflow)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(result => {
                this.dryRunMaxDatas = result.nb_runs_to_analyze;
                this.dryRunModalVisible = true;
                this._cd.markForCheck();
            });
    }

    onSubmitWorkflowUpdate() {
        this.loading = true;
        let actions = [];
        if (this.runnumber !== this.originalRunNumber) {
            actions.push(this._workflowRunService.updateRunNumber(this.project.key, this.workflow, this.runnumber));
        }
        if (this.selectedTagsPurge) {
            this._workflow.purge_tags = this.selectedTagsPurge;
        }

        if (!this._workflow.purge_tags || this._workflow.purge_tags.length === 0) {
            delete this._workflow.purge_tags;
        }

        actions.push(this._store.dispatch(new UpdateWorkflow({
            projectKey: this.project.key,
            workflowName: this.oldName,
            changes: this.workflow
        })));

        forkJoin(...actions)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => {
                if (this.editMode) {
                    this._toast.info('', this._translate.instant('workflow_ascode_updated'));
                } else {
                    this._toast.success('', this._translate.instant('workflow_updated'));
                }
                this._router.navigate([
                    '/project', this.project.key, 'workflow', this.workflow.name
                ], { queryParams: { tab: 'advanced' } });
            });
    }

    deleteWorkflow(): void {
        this._nzModalService.create({
            nzTitle: 'Deleting workflow ' + this._workflow.name,
            nzContent: WorkflowDeleteModalComponent,
            nzFooter: null,
            nzWidth: 900
        });
    }

    addIntegration() {
        this.loading = true;
        let workflowIntegrations = new Array<WorkflowProjectIntegration>();
        let wi = new WorkflowProjectIntegration();
        wi.project_integration = this.selectedIntegration;
        workflowIntegrations.push(wi);
        if (this.workflow.integrations) {
            workflowIntegrations = [wi].concat(this.workflow.integrations);
        }
        this._store.dispatch(new UpdateIntegrationsWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            integrations: workflowIntegrations,
        })).pipe(finalize(() => {
            this.loading = false;
            delete this.selectedIntegration;
            this._cd.markForCheck();
        })).subscribe();
    }

    fileEvent(event: { content: string, file: File }) {
        this.fileTooLarge = event.file.size > 100000;
        if (this.fileTooLarge) {
            return;
        }
        this.iconUpdated = true;
        this._workflow.icon = event.content;
    }

    clickDeleteIntegration(integ: WorkflowProjectIntegration) {
        this.loading = true;
        this._store.dispatch(new DeleteIntegrationWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            projectIntegrationID: integ.project_integration_id
        })).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        })).subscribe();
    }

    filterIntegration(integ: WorkflowProjectIntegration): boolean {
        return !integ.project_integration.model.event;
    }
}
