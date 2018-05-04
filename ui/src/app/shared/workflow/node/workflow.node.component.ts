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
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../decorator/autoUnsubscribe';
import {WorkflowNodeRun, WorkflowRun} from '../../../model/workflow.run.model';
import {PipelineStatus} from '../../../model/pipeline.model';
import {WorkflowNodeRunParamComponent} from './run/node.run.param.component';
import {WorkflowEventStore} from '../../../service/workflow/workflow.event.store';
import {Router} from '@angular/router';

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
    subRun: Subscription;

    zone: NgZone;
    currentNodeRun: WorkflowNodeRun;
    subNodeRun: Subscription;
    pipelineStatus = PipelineStatus;


    warnings = 0;
    loading = false;
    options: {};
    disabled = false;

    isSelected: boolean;
    subSelect: Subscription;


    constructor(private elementRef: ElementRef, private _workflowEventStore: WorkflowEventStore, private _router: Router) {
    }

    ngOnInit(): void {
        this.subSelect = this._workflowEventStore.selectedNode().subscribe(n => {
            if (n && this.node) {
                this.isSelected = this.node.id === n.id;
                return;
            }
            this.isSelected = false;
        });

        this.zone = new NgZone({enableLongStackTrace: false});

        this.subNodeRun = this._workflowEventStore.nodeRunEvents().subscribe(wnr => {
            if (wnr) {
                if (this.workflowRun && wnr.workflow_node_id === this.node.id && this.workflowRun.num === wnr.num) {
                    this.currentNodeRun = wnr;
                }
            }
        });

        this.subRun = this._workflowEventStore.selectedRun().subscribe(wr => {
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

        if (this.workflowRun) {
            this._workflowEventStore.setSelectedNode(this.node, false);
            this._workflowEventStore.setSelectedRun(this.workflowRun);
            this._workflowEventStore.setSelectedNodeRun(this.currentNodeRun, true);
        } else {
            this._workflowEventStore.setSelectedNode(this.node, true);
        }
    }

    goToLogs() {
        if (this._workflowEventStore.isRunSelected() && this.currentNodeRun) {
            this._workflowEventStore.setSelectedNodeRun(this.currentNodeRun);
            this._router.navigate([
                '/project', this.project.key,
                'workflow', this.workflow.name,
                'run', this.currentNodeRun.num,
                'node', this.currentNodeRun.id
            ]);
        } else {
            this._workflowEventStore.setSelectedNode(this.node, true);
            this._router.navigate([
                '/project', this.project.key,
                'pipeline', this.node.pipeline.name
            ], {queryParams: {workflow: this.workflow.name}});
        }
    }
}
