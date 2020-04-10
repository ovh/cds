import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { ActivatedRoute } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Select, Store } from '@ngxs/store';
import { PipelineStatus, SpawnInfo } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WorkflowRun } from 'app/model/workflow.run.model';
import { NotificationService } from 'app/service/notification/notification.service';
import { WorkflowStore } from 'app/service/workflow/workflow.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ProjectState } from 'app/store/project.state';
import { ChangeToRunView, GetWorkflowRun } from 'app/store/workflow.action';
import { WorkflowState } from 'app/store/workflow.state';
import { Observable, Subscription } from 'rxjs';
import { ErrorMessageMap, WarningMessageMap } from './errors';

@Component({
    selector: 'app-workflow-run',
    templateUrl: './workflow.run.html',
    styleUrls: ['./workflow.run.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowRunComponent implements OnInit {

    project: Project;

    @Select(WorkflowState.getSelectedWorkflowRun()) workflowRun$: Observable<WorkflowRun>;
    subWorkflowRun: Subscription;

    workflowName: string;
    version: string;
    direction: string;

    paramsSub: Subscription;

    pipelineStatusEnum = PipelineStatus;
    notificationSubscription: Subscription;
    dataSubs: Subscription;
    paramsSubs: Subscription;
    loadingRun = true;
    warningsMap = WarningMessageMap;
    errorsMap = ErrorMessageMap;
    warnings: Array<SpawnInfo>;
    displayError = false;

    // id, status, workflows, infos, num
    workflowRunData: {};

    constructor(
        private _store: Store,
        private _activatedRoute: ActivatedRoute,
        private _workflowStore: WorkflowStore,
        private _notification: NotificationService,
        private _translate: TranslateService,
        private _titleService: Title,
        private _cd: ChangeDetectorRef
    ) {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
        this.workflowName = this._store.selectSnapshot(WorkflowState.workflowSnapshot).name;
        this._store.dispatch(new ChangeToRunView({}));

        this.paramsSub = this._activatedRoute.params.subscribe(p => {
            this.workflowRunData = {};
            this._cd.markForCheck();
            this._store.dispatch(
                new GetWorkflowRun({
                    projectKey: this.project.key,
                    workflowName: this.workflowName,
                    num: p['number']
                }));
        });


        // Subscribe to workflow Run
        this.subWorkflowRun = this.workflowRun$.subscribe(wr => {
            if (!wr || wr.status === 'Pending') {
                return;
            }

            if (wr && this.workflowRunData && this.workflowRunData['id'] === wr.id && this.workflowRunData['status'] === wr.status) {
                return;
            }

            if (!this.workflowRunData) {
                this.workflowRunData = {};
            }

            // If workflow run change, refresh workflow
            if (wr && this.workflowRunData['id'] !== wr.id) {
                this.workflowRunData['workflow'] = wr.workflow;
                this.workflowName = this._store.selectSnapshot(WorkflowState.workflowSnapshot).name;
            }

            if (wr && this.workflowRunData['id'] && this.workflowRunData['id'] === wr.id
                && this.workflowRunData['status'] !== wr.status && PipelineStatus.isDone(wr.status)) {
                this.handleNotification(wr);
            }

            if (wr && wr.infos && wr.infos.length > 0 && (
                (!this.workflowRunData['infos']) ||
                (this.workflowRunData['infos'] && this.workflowRunData['infos'].length === wr.infos.length)
            )) {
                this.displayError = wr.infos.some((info) => info.type === 'Error');
                this.warnings = wr.infos.filter(i => i.type === 'Warning');
            }

            this.workflowRunData['id'] = wr.id;
            this.workflowRunData['infos'] = wr.infos;
            this.workflowRunData['num'] = wr.num;
            this.workflowRunData['status'] = wr.status;

            this.updateTitle(wr);
            this._cd.markForCheck();
        });
    }

    ngOnInit(): void {
        this.direction = this._workflowStore.getDirection(this.project.key, this.workflowName);
    }

    handleNotification(wr: WorkflowRun) {
        if (wr.num !== parseInt(this._activatedRoute.snapshot.params['number'], 10)) {
            return;
        }

        switch (wr.status) {
            case PipelineStatus.SUCCESS:
                this.notificationSubscription = this._notification.create(this._translate.instant('notification_on_workflow_success', {
                    workflowName: this.workflowName,
                }), {
                    icon: 'assets/images/checked.png',
                    tag: `${this.workflowName}-${wr.num}.${wr.last_subnumber}`
                }).subscribe();
                break;
            case PipelineStatus.FAIL:
                this.notificationSubscription = this._notification.create(this._translate.instant('notification_on_workflow_failing', {
                    workflowName: this.workflowName
                }), {
                    icon: 'assets/images/close.png',
                    tag: `${this.workflowName}-${wr.num}.${wr.last_subnumber}`
                }).subscribe();
                break;
        }
    }

    updateTitle(wr: WorkflowRun) {
        if (!Array.isArray(wr.tags)) {
            return;
        }
        let branch = wr.tags.find((tag) => tag.tag === 'git.branch');
        if (branch) {
            this._titleService.setTitle(`#${wr.num} [${branch.value}] • ${this.workflowName}`);
        }
    }
}
