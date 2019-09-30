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
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WarningModalComponent } from 'app/shared/modal/warning/warning.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { DeleteWorkflow, DeleteWorkflowIcon, UpdateWorkflow, UpdateWorkflowIcon } from 'app/store/workflow.action';
import cloneDeep from 'lodash-es/cloneDeep';
import { DragulaService } from 'ng2-dragula';
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
            if (this._workflow.purge_tags && this._workflow.purge_tags.length) {
                this.purgeTag = this._workflow.purge_tags[0];
            }
        }
    };
    get workflow() { return this._workflow };

    oldName: string;

    runnumber: number;
    originalRunNumber: number;

    allTags = new Array<string>();
    existingTags = new Array<string>();
    selectedTags = new Array<string>();
    purgeTag: string;
    iconUpdated = false;
    tagsToAdd = new Array<string>();

    @ViewChild('updateWarning', { static: false })
    private warningUpdateModal: WarningModalComponent;

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
    ) {
        this._dragularService.createGroup('bag-tag', {
            accepts: function (el, target, source, sibling) {
                if (sibling === null) {
                    return false;
                }
                return true;
            }
        });

        this.dragulaSubscription = this._dragularService.drop('bag-tag').subscribe(({ el, source }) => {
            setTimeout(() => {
                this.updateTagMetadata();
            });
        });
    }

    ngOnDestroy() {
        this._dragularService.destroy('bag-tag');
    }

    ngOnInit(): void {
        if (!this._workflow.metadata) {
            this._workflow.metadata = new Map<string, string>();
        }
        if (this._workflow.metadata['default_tags']) {
            this.selectedTags = this._workflow.metadata['default_tags'].split(',');
        }

        if (this.project.permission !== 7) {
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
    }

    initExistingtags(): void {
        this.existingTags = [];
        this.allTags.forEach(t => {
            if (this.selectedTags.indexOf(t) === -1) {
                this.existingTags.push(t);
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
                this.selectedTags = new Array();
            }
            this.selectedTags.push(...this.tagsToAdd);
            this.initExistingtags();
        }

        this._workflow.metadata['default_tags'] = this.selectedTags.join(',');
        this.tagsToAdd = [];
    }

    removeFromSelectedTags(ind: number): void {
        this.selectedTags.splice(ind, 1);
        this.initExistingtags();
        this.updateTagMetadata();
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
            this._workflow.purge_tags = [this.purgeTag];

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
                    this._toast.success('', this._translate.instant('workflow_updated'));
                    this._router.navigate([
                        '/project', this.project.key, 'workflow', this.workflow.name
                    ], { queryParams: { tab: 'advanced' } });
                });
        }
    }

    updateRunNumber() {
        this._workflowService.updateRunNumber(this.project.key, this.workflow.name, this.runnumber)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            })).subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_updated'));
                this._router.navigate([
                    '/project', this.project.key, 'workflow', this.workflow.name
                ], { queryParams: { tab: 'advanced' } });
            })
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
