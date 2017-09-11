import {AfterViewInit, Component, ElementRef, EventEmitter, Input, NgZone, OnInit, Output, ViewChild} from '@angular/core';
import {Workflow, WorkflowNode, WorkflowNodeJoin, WorkflowNodeTrigger} from '../../../model/workflow.model';
import {Project} from '../../../model/project.model';
import {WorkflowTriggerComponent} from '../trigger/workflow.trigger.component';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {TranslateService} from 'ng2-translate';
import {ToastService} from '../../toast/ToastService';
import {SemanticModalComponent} from 'ng-semantic';
import {WorkflowDeleteNodeComponent} from './delete/workflow.node.delete.component';
import {WorkflowNodeContextComponent} from './context/workflow.node.context.component';
import {cloneDeep} from 'lodash';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../decorator/autoUnsubscribe';
import {PipelineStore} from '../../../service/pipeline/pipeline.store';
import {CDSWorker} from '../../worker/worker';
import {WorkflowNodeRun, WorkflowRun} from '../../../model/workflow.run.model';
import {Router} from '@angular/router';
import {PipelineStatus} from '../../../model/pipeline.model';

declare var _: any;

@Component({
    selector: 'app-workflow-node',
    templateUrl: './workflow.node.html',
    styleUrls: ['./workflow.node.scss']
})
@AutoUnsubscribe()
export class WorkflowNodeComponent implements AfterViewInit, OnInit {

    @Input() node: WorkflowNode;
    @Input() workflow: Workflow;
    @Input() project: Project;

    @Output() linkJoinEvent = new EventEmitter<WorkflowNode>();

    @ViewChild('workflowTrigger')
    workflowTrigger: WorkflowTriggerComponent;
    @ViewChild('workflowDeleteNode')
    workflowDeleteNode: WorkflowDeleteNodeComponent;
    @ViewChild('workflowContext')
    workflowContext: WorkflowNodeContextComponent;

    newTrigger: WorkflowNodeTrigger = new WorkflowNodeTrigger();
    editableNode: WorkflowNode;

    pipelineSubscription: Subscription;
    webworker: CDSWorker;

    zone: NgZone;
    currentNodeRun: WorkflowNodeRun;
    pipelineStatus = PipelineStatus;


    loading = false;
    options: {};
    disabled = false;

    constructor(private elementRef: ElementRef, private _workflowStore: WorkflowStore, private _translate: TranslateService,
                private _toast: ToastService, private _pipelineStore: PipelineStore, private _router: Router) {
    }

    ngOnInit(): void {

        this.zone = new NgZone({enableLongStackTrace: false});
        if (this.webworker) {
            this.webworker.response().subscribe(wrString => {
                let wr = <WorkflowRun>JSON.parse(wrString);
                if (wr.nodes[this.node.id] && wr.nodes[this.node.id].length > 0) {
                    this.currentNodeRun = wr.nodes[this.node.id][0];
                }
            });
        } else {
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

    displayDropdown(): void {
        this.elementRef.nativeElement.style.zIndex = 50;
    }

    ngAfterViewInit() {
        this.elementRef.nativeElement.style.position = 'fixed';
        this.elementRef.nativeElement.style.top = 0;
    }

    openTriggerModal(): void {
        this.newTrigger = new WorkflowNodeTrigger();
        this.newTrigger.workflow_node_id = this.node.id;
        this.workflowTrigger.show({observable: true, closable: false, autofocus: false});
    }

    openDeleteNodeModal(): void {
        this.workflowDeleteNode.show({observable: true, closable: false, autofocus: false});
    }

    openEditContextModal(): void {
        let sub = this.pipelineSubscription = this._pipelineStore.getPipelines(this.project.key, this.node.pipeline.name).subscribe(pips => {
            if (pips.get(this.project.key + '-' + this.node.pipeline.name)) {
                setTimeout(() => {
                    this.workflowContext.show({observable: true, closable: false, autofocus: false});
                    sub.unsubscribe();
                }, 100);
            }
        });
    }

    saveTrigger(): void {
        let clonedWorkflow: Workflow = cloneDeep(this.workflow);
        let currentNode: WorkflowNode;
        if (clonedWorkflow.root.id === this.node.id) {
            currentNode = clonedWorkflow.root;
        } else if (clonedWorkflow.root.triggers) {
            currentNode = Workflow.getNodeByID(this.node.id, clonedWorkflow);
        }

        if (!currentNode) {
            return;
        }

        if (!currentNode.triggers) {
            currentNode.triggers = new Array<WorkflowNodeTrigger>();
        }
        currentNode.triggers.push(cloneDeep(this.newTrigger));
        this.updateWorkflow(clonedWorkflow, this.workflowTrigger.modal);
    }

    updateNode(n: WorkflowNode): void {
        let clonedWorkflow: Workflow = cloneDeep(this.workflow);
        let node = Workflow.getNodeByID(n.id, clonedWorkflow);
        if (!node) {
            return;
        }
        node.context = cloneDeep(n.context);
        delete node.context.application;
        delete node.context.environment;
        this.updateWorkflow(clonedWorkflow, this.workflowContext.nodeContextModal);
    }

    deleteNode(b: boolean): void {
        if (b) {
            let clonedWorkflow: Workflow = cloneDeep(this.workflow);
            if (clonedWorkflow.root.id === this.node.id) {
                this.deleteWorkflow(clonedWorkflow, this.workflowDeleteNode.modal);
            } else {
                clonedWorkflow.root.triggers.forEach((t, i) => {
                    this.removeNode(this.node.id, t.workflow_dest_node, clonedWorkflow.root, i);
                });
                if (clonedWorkflow.joins) {
                    clonedWorkflow.joins.forEach(j => {
                        if (j.triggers) {
                            j.triggers.forEach((t, i) => {
                                this.removeNodeFromJoin(this.node.id, t.workflow_dest_node, j, i);
                            });
                        }
                    });
                }

                this.updateWorkflow(clonedWorkflow, this.workflowDeleteNode.modal);
            }
        }
    }

    removeNodeFromJoin(id: number, node: WorkflowNode, parent: WorkflowNodeJoin, index: number) {
        if (node.id === id) {
            parent.triggers.splice(index, 1);
        }
        if (node.triggers) {
            node.triggers.forEach((t, i) => {
                this.removeNode(id, t.workflow_dest_node, node, i);
            });
        }
    }

    removeNode(id: number, node: WorkflowNode, parent: WorkflowNode, index: number) {
        if (node.id === id) {
            parent.triggers.splice(index, 1);
        }
        if (node.triggers) {
            node.triggers.forEach((t, i) => {
                this.removeNode(id, t.workflow_dest_node, node, i);
            });
        }
    }

    deleteWorkflow(w: Workflow, modal: SemanticModalComponent): void {
        this._workflowStore.deleteWorkflow(this.project.key, w).subscribe(() => {
            this._toast.success('', this._translate.instant('workflow_deleted'));
            modal.hide();
        });
    }

    updateWorkflow(w: Workflow, modal?: SemanticModalComponent): void {
        this.loading = true;
        this._workflowStore.updateWorkflow(this.project.key, w).subscribe(() => {
            this.loading = false;
            this._toast.success('', this._translate.instant('workflow_updated'));
            if (modal) {
                modal.hide();
            }
        }, () => {
            this.loading = false;
        });
    }

    createJoin(): void {
        if (!this.node.ref) {
            this.node.ref = this.node.id.toString();
        }
        let clonedWorkflow: Workflow = cloneDeep(this.workflow);
        if (!clonedWorkflow.joins) {
            clonedWorkflow.joins = new Array<WorkflowNodeJoin>();
        }
        let j = new WorkflowNodeJoin();
        j.source_node_ref.push(this.node.ref);
        clonedWorkflow.joins.push(j);
        this.updateWorkflow(clonedWorkflow);
    }

    linkJoin(): void {
        this.linkJoinEvent.emit(this.node);
    }

    goToNodeRun(): void {
        if (!this.webworker || !this.currentNodeRun) {
            return;
        }
        let pip = Workflow.getNodeByID(this.currentNodeRun.workflow_node_id, this.workflow).pipeline.name;
        this._router.navigate([
            '/project', this.project.key,
            'workflow', this.workflow.name,
            'run', this.currentNodeRun.num,
            'node', this.currentNodeRun.id], {queryParams: { name: pip}});
    }
}
