import {
    Component,
    ElementRef,
    Input,
    NgZone,
    OnInit,
    ViewChild
} from '@angular/core';
import {
    Workflow,
    WorkflowNode
} from '../../../model/workflow.model';
import {Project} from '../../../model/project.model';
import {cloneDeep} from 'lodash';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../decorator/autoUnsubscribe';
import {WorkflowNodeRun, WorkflowRun} from '../../../model/workflow.run.model';
import {Router, ActivatedRoute} from '@angular/router';
import {PipelineStatus} from '../../../model/pipeline.model';
import {WorkflowNodeRunParamComponent} from './run/node.run.param.component';
import {WorkflowCoreService} from '../../../service/workflow/workflow.core.service';
import {ApplicationStore} from '../../../service/application/application.store';

@Component({
    selector: 'app-workflow-node',
    templateUrl: './workflow.node.html',
    styleUrls: ['./workflow.node.scss']
})
@AutoUnsubscribe()
export class WorkflowNodeComponent implements OnInit {

    @Input() node: WorkflowNode;
    @Input() workflow: Workflow;
    @Input() project: Project;

    @ViewChild('workflowRunNode')
    workflowRunNode: WorkflowNodeRunParamComponent;

    workflowRun: WorkflowRun;

    zone: NgZone;
    currentNodeRun: WorkflowNodeRun;
    pipelineStatus = PipelineStatus;


    warnings = 0;
    loading = false;
    options: {};
    disabled = false;
    selectedNodeId: number;

    workflowCoreSub: Subscription;
    applicationSub: Subscription;

    constructor(private elementRef: ElementRef, private _router: Router,
                private _workflowCoreService: WorkflowCoreService,
                private _route: ActivatedRoute,
                private _appStore: ApplicationStore) {
        this._route.queryParams.subscribe((qp) => {
            if (qp['selectedNodeId']) {
                this.selectedNodeId = parseInt(qp['selectedNodeId'], 10);
            } else {
                this.selectedNodeId = null;
            }
        });
    }

    ngOnInit(): void {
        this.zone = new NgZone({enableLongStackTrace: false});

        this.workflowCoreSub = this._workflowCoreService.getCurrentWorkflowRun().subscribe(wr => {
            this.warnings = 0;
            if (wr) {
                if (this.workflowRun && this.workflowRun.id !== wr.id) {
                    this.currentNodeRun = null;
                }
                this.workflowRun = wr;
                if (wr.nodes && wr.nodes[this.node.id] && wr.nodes[this.node.id].length > 0) {
                    this.currentNodeRun = wr.nodes[this.node.id][0];
                }
            } else {
                this.workflowRun = null;
            }
            if (this.currentNodeRun && this.currentNodeRun.status === PipelineStatus.SUCCESS) {
                this.computeWarnings();
            }
        });
        if (!this.workflowRun) {
            this.options = {
                'fullTextSearch': true,
                onHide: () => {
                    this.zone.run(() => {
                        this.elementRef.nativeElement.style.zIndex = 0;
                    });
                }
            };
        }

        if (this.node.context.application && this.node.context.application_id !== 0) {
          this.applicationSub = this._appStore.getApplicationResolver(this.project.key, this.node.context.application.name)
            .subscribe((app) => this.node.context.application = app);
        }
    }

    computeWarnings() {
        this.warnings = 0;
        this.currentNodeRun.stages.forEach((stage) => {
            if (Array.isArray(stage.run_jobs)) {
                this.warnings += stage.run_jobs.reduce((fail, job) => {
                    if (!job.job || !Array.isArray(job.job.step_status)) {
                        return fail;
                    }
                    return fail + job.job.step_status.reduce((failStep, step) => {
                        if (step.status === PipelineStatus.FAIL) {
                            return failStep + 1;
                        }
                        return failStep;
                    }, 0);
                }, 0);
            }
        })
    }

    goToNodeRun(): void {
        if (this.workflow.previewMode) {
          return;
        }

        let qps = cloneDeep(this._route.snapshot.queryParams);
        qps['selectedJoinId'] = null;
        qps['selectedHookId'] = null;

        if (!this._route.snapshot.params['number']) {
            qps['selectedNodeRunId'] = null;
            qps['selectedNodeRunNum'] = null;

            this._router.navigate([
                '/project', this.project.key,
                'workflow', this.workflow.name
            ], { queryParams: Object.assign({}, qps, {selectedNodeId: this.node.id })});
        } else {
            qps['selectedJoinId'] = null;
            qps['selectedNodeId'] = null;
            qps['selectedHookId'] = null;
            this._router.navigate([
                '/project', this.project.key,
                'workflow', this.workflow.name,
                'run', this.currentNodeRun ? this.currentNodeRun.num : this._route.snapshot.params['number']], {
                    queryParams: Object.assign({}, qps, {
                        selectedNodeRunId: this.currentNodeRun ? this.currentNodeRun.id : -1,
                        selectedNodeRunNum: this.currentNodeRun ? this.currentNodeRun.num : 0,
                        selectedNodeId: this.currentNodeRun ? this.currentNodeRun.workflow_node_id : this.node.id
                    })
                });
        }
    }

    goToLogs() {
        let pip = this.node.pipeline.name;
        if (this.currentNodeRun) {
            this._router.navigate([
                '/project', this.project.key,
                'workflow', this.workflow.name,
                'run', this.currentNodeRun.num,
                'node', this.currentNodeRun.id], {queryParams: {
                    name: pip,
                    selectedNodeRunId: this.currentNodeRun ? this.currentNodeRun.id : -1,
                    selectedNodeRunNum: this.currentNodeRun ? this.currentNodeRun.num : 0,
                    selectedNodeId: this.currentNodeRun ? this.currentNodeRun.workflow_node_id : this.node.id}});
        } else {
          this._router.navigate([
              '/project', this.project.key,
              'pipeline', pip
          ], {queryParams: {workflow: this.workflow.name}});
        }
    }
}
