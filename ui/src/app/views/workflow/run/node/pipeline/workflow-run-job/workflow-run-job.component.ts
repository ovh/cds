import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, OnInit } from '@angular/core';
import { Store } from '@ngxs/store';
import { Job } from 'app/model/job.model';
import { CDNLogLink } from 'app/model/pipeline.model';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ProjectState } from 'app/store/project.state';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';

export enum DisplayMode {
    ANSI = 'ansi',
    HTML = 'html',
}

export class Tab {
    name: string;
}

export class Step {
    name: string;
    lines: Array<Line>;
    open: boolean;
    link: CDNLogLink;

    clickOpen(): void {
        this.open = !this.open;
    }
}

export class Line {
    display_number: number;
    number: number;
    value: string;
}

@Component({
    selector: 'app-workflow-run-job',
    templateUrl: './workflow-run-job.html',
    styleUrls: ['workflow-run-job.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowRunJobComponent implements OnInit, OnChanges {
    @Input() job: Job;

    mode = DisplayMode.ANSI;
    displayModes = DisplayMode;
    tabs: Array<Tab>;
    currentTabIndex = 0;

    steps: Array<Step>;

    constructor(
        private _cd: ChangeDetectorRef,
        private _store: Store,
        private _workflowService: WorkflowService,
        private _http: HttpClient
    ) { }

    ngOnInit(): void { }

    ngOnChanges(): void {
        if (!this.job) { return; }

        this.tabs = [{ name: 'Job' }];
        this.tabs = this.tabs.concat(this.job.action.requirements.filter(r => r.type === 'service').map(r => <Tab>{ name: r.name }));


        this.steps = this.job.action.actions.map(a => {
            let step = new Step();
            step.name = a.step_name ? a.step_name : a.name;
            return step;
        });

        this.loadStepLinks();

        this._cd.markForCheck();
    }

    selectTab(i: number): void {
        this.currentTabIndex = i;
        this._cd.markForCheck();
    }

    clickMode(mode: DisplayMode): void {
        this.mode = mode;
        this._cd.markForCheck();
    }

    async loadStepLinks() {
        let projectKey = this._store.selectSnapshot(ProjectState.projectSnapshot).key;
        let workflowName = this._store.selectSnapshot(WorkflowState.workflowSnapshot).name;
        let nodeRunID = this._store.selectSnapshot(WorkflowState).workflowNodeRun.id;
        let nodeJobRunID = this._store.selectSnapshot(WorkflowState.getSelectedWorkflowNodeJobRun()).id;

        if (!this.job.step_status) {
            return;
        }

        for (let i = 0; i < this.steps.length; i++) {
            if (!this.job.step_status || !this.job.step_status[i]) { return; }
            this.steps[i].link = await this._workflowService.getStepLink(projectKey, workflowName, nodeRunID, nodeJobRunID, i).toPromise();
            let lines = await this._http.get<Array<Line>>(`./cdscdn${this.steps[i].link.lines_path}`).toPromise();
            this.steps[i].lines = lines.map(l => {
                let line = new Line();
                line.number = l.number;
                line.value = l.value;
                return line;
            });
        }

        this._cd.markForCheck();
    }
}
