import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnInit,
    Output,
    ViewChild
} from '@angular/core';
import {Store} from '@ngxs/store';
import {Parameter} from 'app/model/parameter.model';
import {CDNLine} from 'app/model/pipeline.model';
import {WorkflowNodeJobRun} from 'app/model/workflow.run.model';
import {WorkflowRunService} from 'app/service/services.module';
import {AutoUnsubscribe} from 'app/shared/decorator/autoUnsubscribe';
import {Tab} from 'app/shared/tabs/tabs.component';
import {
    ScrollTarget,
    WorkflowRunJobComponent
} from 'app/views/workflow/run/node/pipeline/workflow-run-job/workflow-run-job.component';
import {V2WorkflowRun, V2WorkflowRunJob, WorkflowRunInfo} from "app/model/v2.workflow.run.model";


@Component({
    selector: 'app-run-job',
    templateUrl: './run-job.html',
    styleUrls: ['./run-job.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunJobComponent implements OnInit {
    @ViewChild('workflowRunJob') workflowRunJob: WorkflowRunJobComponent;

    @Input() workflowRun: V2WorkflowRun
    @Input() jobRun: V2WorkflowRunJob;
    @Input() jobRunInfos: Array<WorkflowRunInfo>;

    @Output() onClickClose = new EventEmitter<void>();

    tabs: Array<Tab>;
    selectedTab: Tab;
    loading = false;
    selectedNodeJobRun: WorkflowNodeJobRun;
    variables: { [key: string]: Array<Parameter> } = {};
    variableKeys: Array<string> = [];

    constructor(
        private _cd: ChangeDetectorRef,
        private _workflowRunService: WorkflowRunService,
        private _store: Store
    ) {
        this.tabs = [<Tab>{
            title: 'Logs',
            key: 'logs',
            default: true
        }, <Tab>{
            title: 'Problems',
            icon: 'warning',
            iconTheme: 'fill',
            key: 'problems',
        }, <Tab>{
            title: 'Infos',
            key: 'infos',
            icon: 'info-circle',
            iconTheme: 'outline',
        }];
    }

    ngOnInit(): void {
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
            if (name.indexOf('cds.proj.', 0) === 0) {
                return 'project';
            }
            if (name.indexOf('cds.app.', 0) === 0) {
                return 'application';
            }
            if (name.indexOf('cds.pip.', 0) === 0) {
                return 'pipeline';
            }
            if (name.indexOf('cds.env.', 0) === 0) {
                return 'environment';
            }
            if (name.indexOf('cds.parent.', 0) === 0) {
                return 'parent';
            }
            if (name.indexOf('cds.build.', 0) === 0) {
                return 'build';
            }
            if (name.indexOf('git.', 0) === 0) {
                return 'git';
            }
            if (name.indexOf('workflow.', 0) === 0) {
                return 'workflow';
            }
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
