import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnChanges,
    OnDestroy,
    Output,
    ViewChild
} from '@angular/core';
import { Parameter } from 'app/model/parameter.model';
import { CDNLine, CDNStreamFilter, PipelineStatus } from 'app/model/pipeline.model';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Tab } from 'app/shared/tabs/tabs.component';
import { V2WorkflowRun, V2WorkflowRunJob, WorkflowRunInfo } from "app/model/v2.workflow.run.model";
import { RunJobLogsComponent } from "./run-job-logs.component";
import { WebSocketSubject, webSocket } from 'rxjs/webSocket';
import { Subscription, delay, retryWhen } from 'rxjs';
import { Router } from '@angular/router';


@Component({
    selector: 'app-run-job',
    templateUrl: './run-job.html',
    styleUrls: ['./run-job.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunJobComponent implements OnChanges, OnDestroy {
    @ViewChild('runJobLogs') runJobLogs: RunJobLogsComponent;

    @Input() workflowRun: V2WorkflowRun
    @Input() jobRun: V2WorkflowRunJob;
    @Input() jobRunInfos: Array<WorkflowRunInfo>;
    @Output() onClickClose = new EventEmitter<void>();

    defaultTabs: Array<Tab>;
    tabs: Array<Tab>;
    selectedTab: Tab;
    loading = false;
    selectedNodeJobRun: WorkflowNodeJobRun;
    variables: { [key: string]: Array<Parameter> } = {};
    variableKeys: Array<string> = [];
    websocket: WebSocketSubject<any>;
    websocketSubscription: Subscription;
    cdnFilter: CDNStreamFilter;

    constructor(
        private _cd: ChangeDetectorRef,
        private _router: Router
    ) {
        this.defaultTabs = [<Tab>{
            title: 'Logs',
            key: 'logs',
            default: true
        }, <Tab>{
            title: 'Infos',
            key: 'infos',
            icon: 'info-circle',
            iconTheme: 'outline',
        }];
        this.tabs = [...this.defaultTabs];
        this.tabs[0].default = true;
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnChanges(): void {
        if (this.jobRunInfos && !!this.jobRunInfos.find(i => i.level === 'warning' || i.level === 'error')) {
            this.tabs = [<Tab>{
                title: 'Problems',
                icon: 'warning',
                iconTheme: 'fill',
                key: 'problems',
                default: true,
            }, ...this.defaultTabs];
        }

        if (this.jobRun && !PipelineStatus.isDone(this.jobRun.status) && !this.websocketSubscription) {
            this.startStreamingLogsForJob();
        }

        if (this.jobRun && PipelineStatus.isDone(this.jobRun.status) && this.websocketSubscription) {
            this.websocketSubscription.unsubscribe();
        }
    }

    selectTab(tab: Tab): void {
        this.selectedTab = tab;
        if (this.selectedTab.key === 'variables') {
            this.setVariables(this.selectedNodeJobRun.parameters);
        }
    }

    onJobScroll(target) {

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

    startStreamingLogsForJob() {
        if (!this.cdnFilter) {
            this.cdnFilter = new CDNStreamFilter();
        }
        if (this.cdnFilter.job_run_id === this.jobRun.id) {
            return;
        }

        if (!this.websocket) {
            const protocol = window.location.protocol.replace('http', 'ws');
            const host = window.location.host;
            const href = this._router['location']._basePath;
            this.websocket = webSocket({
                url: `${protocol}//${host}${href}/cdscdn/v2/item/stream`,
                openObserver: {
                    next: value => {
                        if (value.type === 'open') {
                            this.cdnFilter.job_run_id = this.jobRun.id;
                            this.websocket.next(this.cdnFilter);
                        }
                    }
                }
            });

            this.websocketSubscription = this.websocket
                .pipe(retryWhen(errors => errors.pipe(delay(2000))))
                .subscribe((l: CDNLine) => {
                    if (this.runJobLogs) {
                        this.runJobLogs.receiveLogs(l);
                    } else {
                        console.log('job component not loaded');
                    }
                }, (err) => {
                    console.error('Error: ', err);
                }, () => {
                    console.warn('Websocket Completed');
                });
        } else {
            // Refresh cdn filter if job changed
            if (this.cdnFilter.job_run_id !== this.jobRun.id) {
                this.cdnFilter.job_run_id = this.jobRun.id;
                this.websocket.next(this.cdnFilter);
            }
        }
    }
}
