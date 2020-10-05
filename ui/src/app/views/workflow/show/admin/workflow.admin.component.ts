import {
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
import { ModalTemplate, SuiActiveModal, SuiModalService, SuiPopup, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { Project } from 'app/model/project.model';
import { PurgeRunToDelete } from 'app/model/purge.model';
import { Workflow } from 'app/model/workflow.model';
import { ThemeStore } from 'app/service/theme/theme.store';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WarningModalComponent } from 'app/shared/modal/warning/warning.component';
import { Column, ColumnType } from 'app/shared/table/data-table.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { FeatureState } from 'app/store/feature.state';
import { DeleteWorkflow, DeleteWorkflowIcon, UpdateWorkflow, UpdateWorkflowIcon } from 'app/store/workflow.action';
import cloneDeep from 'lodash-es/cloneDeep';
import { CodemirrorComponent } from 'ng2-codemirror-typescript/Codemirror';
import { DragulaService } from 'ng2-dragula-sgu';
import { forkJoin, Subscription } from 'rxjs';
import { finalize, first } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-admin',
    templateUrl: 'workflow.admin.component.html',
    styleUrls: ['./workflow.admin.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowAdminComponent implements OnInit, OnDestroy {

    @Input() project: Project;

    _workflow: Workflow;
    @Input('workflow')
    set workflow(data: Workflow) {
        if (data) {
            this._workflow = cloneDeep(data);
        }
    }
    get workflow() { return this._workflow }

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
    workflowRetentionPolicyEnabled = false;
    codeMirrorConfig: any;

    @ViewChild('updateWarning')
    private warningUpdateModal: WarningModalComponent;
    @ViewChild('codemirrorRetentionPolicy') codemirror: CodemirrorComponent;
    themeSubscription: Subscription;


    dryRunColumns = [];
    dryRunDatas: Array<PurgeRunToDelete>;
    @ViewChild('modalDryRun') dryRunModal: ModalTemplate<boolean, boolean, void>;
    modal: SuiActiveModal<boolean, boolean, void>;

    loading = false;
    fileTooLarge = false;
    dragulaSubscription: Subscription;

    constructor(
        private store: Store,
        public _translate: TranslateService,
        private _toast: ToastService,
        private _router: Router,
        private _workflowRunService: WorkflowRunService,
        private _workflowService: WorkflowService,
        private _cd: ChangeDetectorRef,
        private _dragularService: DragulaService,
        private _theme: ThemeStore,
        private _modalService: SuiModalService
    ) {
        this._dragularService.createGroup('bag-tag', {
            accepts: function (el, target, source, sibling) {
                return sibling !== null;
            }
        });

        this.dragulaSubscription = this._dragularService.drop('bag-tag').subscribe(({}) => {
            setTimeout(() => {
                this.updateTagMetadata();
            });
        });
        this.dryRunColumns = [
            <Column<PurgeRunToDelete>>{
                name: 'run_number',
                class: 'two',
                selector: (r: PurgeRunToDelete) => r.num
            },
            <Column<PurgeRunToDelete>>{
                type: ColumnType.TEXT,
                name: 'status',
                class: 'two',
                selector: (r: PurgeRunToDelete) => r.status
            }
        ];
    }

    ngOnDestroy() {
        this._dragularService.destroy('bag-tag');
    }

    ngOnInit(): void {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'text/x-lua',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true,
            readOnly: !this.editMode,
            gutters: ['CodeMirror-lint-markers'],
        };

        this.themeSubscription = this._theme.get().subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
                this._cd.markForCheck();
            }
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

        let featResult = this.store.selectSnapshot(FeatureState.featureProject('workflow-retention-policy',
            JSON.stringify({ 'project_key': this.project.key })))
        this.workflowRetentionPolicyEnabled = featResult?.enabled;
        this._cd.markForCheck();
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

    deleteIcon(): void {
        this.loading = true;
        this.store.dispatch(new DeleteWorkflowIcon({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => this._toast.success('', this._translate.instant('workflow_updated')));
    }

    updateIcon(): void {
        this.loading = true;
        this.store.dispatch(new UpdateWorkflowIcon({
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
        this._workflowService.retentionPolicyDryRun(this.workflow).subscribe(wr => {
            this.dryRunDatas = wr;
            const config = new TemplateModalConfig<boolean, boolean, void>(this.dryRunModal);
            config.mustScroll = true;
            this.modal = this._modalService.open(config);
            this._cd.markForCheck();
        });
    }

    onSubmitWorkflowUpdate(skip?: boolean) {
        if (!skip && this.workflow.externalChange) {
            this.warningUpdateModal.show();
        } else {
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

            actions.push(this.store.dispatch(new UpdateWorkflow({
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
    }

    deleteWorkflow(): void {
        this.store.dispatch(new DeleteWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_deleted'));
                this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'workflows' } });
            });
    }

    fileEvent(event: { content: string, file: File }) {
        this.fileTooLarge = event.file.size > 100000;
        if (this.fileTooLarge) {
            return;
        }
        this.iconUpdated = true;
        this._workflow.icon = event.content;
    }
}
