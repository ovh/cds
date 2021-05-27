import { ChangeDetectionStrategy, ChangeDetectorRef, Component, ElementRef, EventEmitter, Input, OnChanges, Output, ViewChild } from '@angular/core';
import { Parameter } from 'app/model/parameter.model';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { WorkflowRunService } from 'app/service/services.module';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Tab } from 'app/shared/tabs/tabs.component';
import { ScrollTarget } from 'app/views/workflow/run/node/pipeline/workflow-run-job/workflow-run-job.component';
import { finalize } from 'rxjs/operators';
import { JobRun } from '../workflowv3.model';

@Component({
    selector: 'app-workflowv3-run-job',
    templateUrl: './workflowv3-run-job.html',
    styleUrls: ['./workflowv3-run-job.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowV3RunJobComponent implements OnChanges {
    @ViewChild('workflowRunJob', { read: ElementRef }) workflowRunJob: ElementRef;

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

    constructor(
        private _cd: ChangeDetectorRef,
        private _workflowRunService: WorkflowRunService
    ) {
        this.tabs = [<Tab>{
            translate: 'common_logs',
            key: 'logs',
            default: true
        }, <Tab>{
            translate: 'common_variables',
            key: 'variables'
        }];
    }

    ngOnChanges(): void {
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
        this.workflowRunJob.nativeElement.children[0].scrollTop = target === ScrollTarget.TOP ?
            0 : this.workflowRunJob.nativeElement.children[0].scrollHeight;
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
        }

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
}
