import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { ActivatedRoute } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { PipelineStatus, SpawnInfo } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WorkflowRun } from 'app/model/workflow.run.model';
import { NotificationService } from 'app/service/notification/notification.service';
import { WorkflowStore } from 'app/service/workflow/workflow.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import { ChangeToRunView, GetWorkflowRun } from 'app/store/workflow.action';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import { Subscription } from 'rxjs';
import { filter } from 'rxjs/operators';
import { ErrorMessageMap, WarningMessageMap } from './errors';

@Component({
    selector: 'app-workflow-run',
    templateUrl: './workflow.run.html',
    styleUrls: ['./workflow.run.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowRunComponent implements OnInit {

    subWorkflow: Subscription;

    project: Project;
    project$: Subscription;

    workflowName: string;
    version: string;
    direction: string;

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
        // Get project
        this.dataSubs = this._activatedRoute.data.subscribe(datas => {
            if (!this.project || (<Project>datas['project']).key !== this.project.key) {
                this.project = datas['project'];
                this.workflowRunData = null;
                this.workflowName = '';
            }
        });

        this.project$ = this._store.select(ProjectState)
            .pipe(filter((prj) => prj != null))
            .subscribe((projState: ProjectStateModel) => this.project = projState.project);

        this.workflowName = this._activatedRoute.snapshot.parent.params['workflowName'];
        this._store.dispatch(new ChangeToRunView({}));

        // Subscribe to route event
        this.paramsSubs = this._activatedRoute.params.subscribe(ps => {
            this._cd.markForCheck();
            // if there is no current workflow run
            if (!this.workflowRunData) {
                this._store.dispatch(
                    new GetWorkflowRun({
                        projectKey: this.project.key,
                        workflowName: this.workflowName,
                        num: ps['number']
                    }));
            } else {
                if (this.workflowRunData['workflow'].name !== this.workflowName || this.workflowRunData['num'] !== ps['number']) {
                    this._store.dispatch(
                        new GetWorkflowRun({
                            projectKey: this.project.key,
                            workflowName: this.workflowName,
                            num: ps['number']
                        }));
                }
            }
        });

        this.subWorkflow = this._store.select(WorkflowState.getCurrent()).subscribe((s: WorkflowStateModel) => {
            this.loadingRun = s.loadingWorkflowRun;
            if (s.workflowRun) {
                if (!this.workflowRunData) {
                    this.workflowRunData = {};
                }
                if (!this.workflowRunData['workflow'] || !this.workflowRunData['workflow'].workflow_data) {
                    this.workflowRunData['workflow'] = s.workflowRun.workflow;
                    this.workflowName = s.workflowRun.workflow.name;
                }

                if (this.workflowRunData['id'] && this.workflowRunData['id'] === s.workflowRun.id
                    && this.workflowRunData['status'] !== s.workflowRun.status &&
                    PipelineStatus.isDone(s.workflowRun.status)) {
                    this.handleNotification(s.workflowRun);
                }
                this.workflowRunData['id'] = s.workflowRun.id;
                this.workflowRunData['status'] = s.workflowRun.status;
                this.workflowRunData['infos'] = s.workflowRun.infos;
                this.workflowRunData['num'] = s.workflowRun.num;

                if (s.workflowRun.infos && s.workflowRun.infos.length > 0) {
                    this.displayError = s.workflowRun.infos.some((info) => info.type === 'Error');
                    this.warnings = s.workflowRun.infos.filter(i => i.type === 'Warning');
                }

                this.updateTitle(s.workflowRun);
                this._cd.markForCheck();
            } else {
                delete this.workflowRunData;
            }
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
