import {WorkflowNode} from '../../../model/workflow.model';
import {Component, NgZone, OnDestroy, OnInit, ViewChild} from '@angular/core';
import {ActivatedRoute} from '@angular/router';
import {NotificationService} from '../../../service/notification/notification.service';
import {Project} from '../../../model/project.model';
import {CDSWorker} from '../../../shared/worker/worker';
import {WorkflowRun} from '../../../model/workflow.run.model';
import {PipelineStatus} from '../../../model/pipeline.model';
import {environment} from '../../../../environments/environment';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {WorkflowNodeRunParamComponent} from '../../../shared/workflow/node/run/node.run.param.component';
import {WorkflowCoreService} from '../../../service/workflow/workflow.core.service';
import {cloneDeep} from 'lodash';
import {TranslateService} from '@ngx-translate/core';

@Component({
    selector: 'app-workflow-run',
    templateUrl: './workflow.run.html',
    styleUrls: ['./workflow.run.scss']
})
@AutoUnsubscribe()
export class WorkflowRunComponent implements OnDestroy, OnInit {
    @ViewChild('workflowNodeRunParam')
    runWithParamComponent: WorkflowNodeRunParamComponent;

    project: Project;
    runWorkflowWorker: CDSWorker;
    runSubsription: Subscription;
    workflowRun: WorkflowRun;
    tmpWorkflowRun: WorkflowRun;
    zone: NgZone;
    workflowName: string;
    version: string;
    direction: string;
    currentNumber: number;

    nodeToRun: WorkflowNode;

    pipelineStatusEnum = PipelineStatus;
    notificationSubscription: Subscription;
    workflowCoreSub: Subscription;
    loadingRun = false;

    constructor(private _activatedRoute: ActivatedRoute, private _authStore: AuthentificationStore,
                private _workflowStore: WorkflowStore,
                private _workflowCoreService: WorkflowCoreService, private _notification: NotificationService,
                private _translate: TranslateService) {
        this.zone = new NgZone({enableLongStackTrace: false});

        // Update data if route change
        this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this._activatedRoute.parent.params.subscribe(params => {
            this.workflowName = params['workflowName'];
        });
        this._activatedRoute.queryParams.subscribe(p => {
            if (this.workflowRun && p['subnum']) {
                this.startWorker(this.workflowRun.num);
            }
        });
        this._activatedRoute.params.subscribe(params => {
            let number = params['number'];
            if (this.project.key && this.workflowName && number && number !== this.currentNumber) {
                this.currentNumber = number;
                this.startWorker(number);
            }
        });

        this.workflowCoreSub = this._workflowCoreService.getCurrentWorkflowRun().subscribe((wr) => {
            if (this.workflowRun && wr && wr.id !== this.workflowRun.id) {
                if (wr.num !== this.currentNumber) {
                    this.currentNumber = wr.num;
                    this.workflowRun = wr;
                    this.startWorker(wr.num);
                }
            }
        });
    }

    startWorker(num: number): void {
        this.loadingRun = true;
        // Start web worker
        if (this.runWorkflowWorker) {
            this.runWorkflowWorker.stop();
            this.runWorkflowWorker = null;
        }
        this.runWorkflowWorker = new CDSWorker('./assets/worker/web/workflow2.js');
        this.runWorkflowWorker.start({
            'user': this._authStore.getUser(),
            'session': this._authStore.getSessionToken(),
            'api': environment.apiURL,
            key: this.project.key,
            workflowName: this.workflowName,
            number: num
        });
        this.runSubsription = this.runWorkflowWorker.response().subscribe(wrString => {
            if (wrString) {
                this.zone.run(() => {
                    this.loadingRun = false;
                    this.workflowRun = <WorkflowRun>JSON.parse(wrString);
                    this._workflowCoreService.setCurrentWorkflowRun(this.workflowRun);
                    if (this.workflowRun.status === PipelineStatus.STOPPED ||
                        this.workflowRun.status === PipelineStatus.FAIL || this.workflowRun.status === PipelineStatus.SUCCESS) {
                        this.runWorkflowWorker.stop();
                        this.runSubsription.unsubscribe();
                        if (this.tmpWorkflowRun != null && this.tmpWorkflowRun.id === this.workflowRun.id &&
                            this.tmpWorkflowRun.status !== PipelineStatus.STOPPED && this.tmpWorkflowRun.status !== PipelineStatus.FAIL &&
                            this.tmpWorkflowRun.status !== PipelineStatus.SUCCESS) {
                            this.handleNotification();
                        }
                    }

                    this.tmpWorkflowRun = this.workflowRun;
                });
            }
        });
    }

    handleNotification() {
        switch (this.workflowRun.status) {
            case PipelineStatus.SUCCESS:
                this.notificationSubscription = this._notification.create(this._translate.instant('notification_on_workflow_success', {
                    workflowName: this.workflowName,
                }), {icon: 'assets/images/checked.png'}).subscribe();
                break;
            case PipelineStatus.FAIL:
                this.notificationSubscription = this._notification.create(this._translate.instant('notification_on_workflow_failing', {
                    workflowName: this.workflowName
                }), {icon: 'assets/images/close.png'}).subscribe();
                break;
        }
    }

    relaunch() {
        if (this.runWithParamComponent && this.runWithParamComponent.show) {
            let rootNodeRun = this.workflowRun.nodes[this.workflowRun.workflow.root.id][0];
            this.nodeToRun = cloneDeep(this.workflowRun.workflow.root);
            if (rootNodeRun.hook_event) {
                this.nodeToRun.context.default_payload = rootNodeRun.hook_event.payload;
                this.nodeToRun.context.default_pipeline_parameters = rootNodeRun.hook_event.pipeline_parameter;
            }
            if (rootNodeRun.manual) {
                this.nodeToRun.context.default_payload = rootNodeRun.manual.payload;
                this.nodeToRun.context.default_pipeline_parameters = rootNodeRun.manual.pipeline_parameter;
            }
            setTimeout(() => this.runWithParamComponent.show());
        }
    }

    ngOnDestroy(): void {
        if (this.runWorkflowWorker) {
            this.runWorkflowWorker.stop();
            this.runWorkflowWorker = null;
        }
    }

    ngOnInit(): void {
        this.direction = this._workflowStore.getDirection(this.project.key, this.workflowName);
    }
}
