import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, OnInit, Output, ViewChild } from '@angular/core';
import { Store } from '@ngxs/store';
import { EventType } from 'app/model/event.model';
import { Parameter } from 'app/model/parameter.model';
import { CDNLine } from 'app/model/pipeline.model';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { WorkflowRunService } from 'app/service/services.module';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Tab } from 'app/shared/tabs/tabs.component';
import { EventState } from 'app/store/event.state';
import {
    ScrollTarget, WorkflowRunJobComponent
} from 'app/views/workflow/run/node/pipeline/workflow-run-job/workflow-run-job.component';
import { Subscription, timer } from 'rxjs';
import { debounce, filter, finalize } from 'rxjs/operators';
import { JobRun } from '../workflowv3.model';

@Component({
    selector: 'app-workflowv3-run-job',
    templateUrl: './workflowv3-run-job.html',
    styleUrls: ['./workflowv3-run-job.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowV3RunJobComponent implements OnInit, OnChanges {
    @ViewChild('workflowRunJob') workflowRunJob: WorkflowRunJobComponent;

    @Input() projectKey: string;
    @Input() workflowName: string;
    @Input() workflowRunNum: number;
    @Input() jobRun: JobRun;

    @Output() onClickClose = new EventEmitter<void>();

    tabs: Array<Tab>;
    selectedTab: Tab;
    loading = false;
    selectedNodeJobRun: WorkflowNodeJobRun;
    variables: { [key: string]: Array<Parameter> } = {};
    variableKeys: Array<string> = [];
    eventSubscription: Subscription;

    constructor(
        private _cd: ChangeDetectorRef,
        private _workflowRunService: WorkflowRunService,
        private _store: Store
    ) {
        this.tabs = [<Tab>{
            title: 'common_logs',
            key: 'logs',
            default: true
        }, <Tab>{
            title: 'common_variables',
            key: 'variables'
        }];
    }

    ngOnInit(): void {
        // Refresh workflow node job run when receiving new events
        this.eventSubscription = this._store.select(EventState.last)
            .pipe(
                filter(e => e && this.jobRun && e.type_event === EventType.RUN_WORKFLOW_NODE
                    && e.project_key === this.projectKey
                    && e.workflow_name === this.workflowName
                    && e.workflow_run_num === this.workflowRunNum
                    && e.workflow_node_run_id === this.jobRun.workflow_node_run_id),
                debounce(() => timer(500))
            )
            .subscribe(e => {
                this.loadNodeJobRun();
            });
    }

    ngOnChanges(): void {
        this.loadNodeJobRun();
    }

    loadNodeJobRun(): void {
        this.loading = true;
        this._workflowRunService.getWorkflowNodeRun(this.projectKey, this.workflowName,
            this.workflowRunNum, this.jobRun.workflow_node_run_id)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(wnr => {
                for (let i = 0; i < wnr.stages.length; i++) {
                    this.selectedNodeJobRun = wnr.stages[i].run_jobs.find(rj => rj.id === this.jobRun.workflow_node_job_run_id);
                    if (this.selectedNodeJobRun) {
                        break;
                    }
                }
            });
    }

    selectTab(tab: Tab): void {
        this.selectedTab = tab;
        if (this.selectedTab.key === 'variables') {
            this.setVariables(this.selectedNodeJobRun.parameters);
        }
    }

    onJobScroll(target: ScrollTarget): void {
        this.workflowRunJob.onJobScroll(target);
    }

    setVariables(data: Array<Parameter>) {
        this.variables = {};
        if (!data) {
            return;
        }

        const computeType = (name: string): string => {
            if (name.indexOf('cds.proj.', 0) === 0) { return 'project'; }
            if (name.indexOf('cds.app.', 0) === 0) { return 'application'; }
            if (name.indexOf('cds.pip.', 0) === 0) { return 'pipeline'; }
            if (name.indexOf('cds.env.', 0) === 0) { return 'environment'; }
            if (name.indexOf('cds.parent.', 0) === 0) { return 'parent'; }
            if (name.indexOf('cds.build.', 0) === 0) { return 'build'; }
            if (name.indexOf('git.', 0) === 0) { return 'git'; }
            if (name.indexOf('workflow.', 0) === 0) { return 'workflow'; }
            return 'cds';
        };

        data.forEach(p => {
            const t = computeType(p.name);
            if (!this.variables[t]) {
                this.variables[t] = [];
            }
            this.variables[t].push(p);
        });

        this.variableKeys = Object.keys(this.variables).sort();

        this._cd.markForCheck();
    }

    clickClose(): void {
        this.onClickClose.emit();
    }

    receiveLogs(l: CDNLine) {
        this.workflowRunJob.receiveLogs(l);
    }
}
