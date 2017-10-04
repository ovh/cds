import { WorkflowNode } from '../../../model/workflow.model';
import {Component, NgZone, OnDestroy, OnInit, ViewChild} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {Project} from '../../../model/project.model';
import {CDSWorker} from '../../../shared/worker/worker';
import {WorkflowRun} from '../../../model/workflow.run.model';
import {PipelineStatus} from '../../../model/pipeline.model';
import {environment} from '../../../../environments/environment';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {WorkflowRunService} from '../../../service/workflow/run/workflow.run.service';
import {WorkflowNodeRunParamComponent} from '../../../shared/workflow/node/run/node.run.param.component';
import {cloneDeep} from 'lodash';

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
    zone: NgZone;
    workflowName: string;
    version: string;
    direction: string;

    nodeToRun: WorkflowNode;

    pipelineStatusEnum = PipelineStatus;

    constructor(private _activatedRoute: ActivatedRoute, private _authStore: AuthentificationStore,
      private _router: Router, private _workflowStore: WorkflowStore, private _workflowRunService: WorkflowRunService) {
        this.zone = new NgZone({enableLongStackTrace: false});

        // Update data if route change
        this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this._activatedRoute.parent.params.subscribe(params => {
            this.workflowName = params['workflowName'];
        });
        this._activatedRoute.params.subscribe(params => {
            let number = params['number'];
            if (this.project.key && this.workflowName && number) {
                // Start web worker
                if (this.runWorkflowWorker) {
                    this.runWorkflowWorker.stop();
                }
                this.runWorkflowWorker = new CDSWorker('./assets/worker/web/workflow2.js');
                this.runWorkflowWorker.start({
                    'user': this._authStore.getUser(),
                    'session': this._authStore.getSessionToken(),
                    'api': environment.apiURL,
                    key: this.project.key,
                    workflowName: this.workflowName,
                    number: number
                });
                this.runSubsription = this.runWorkflowWorker.response().subscribe(wrString => {
                    if (wrString) {
                        this.zone.run(() => {
                            this.workflowRun = <WorkflowRun>JSON.parse(wrString);
                            if (this.workflowRun.status === PipelineStatus.FAIL || this.workflowRun.status === PipelineStatus.SUCCESS) {
                                this.runWorkflowWorker.stop();
                                this.runSubsription.unsubscribe();
                            }
                        });
                    }
                });
            }
        });
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
            this.runWithParamComponent.show();
        }
    }

    ngOnDestroy(): void {
        if (this.runWorkflowWorker) {
            this.runWorkflowWorker.stop();
        }
    }

    ngOnInit(): void {
      this.direction = this._workflowStore.getDirection(this.project.key, this.workflowName);
    }
}
