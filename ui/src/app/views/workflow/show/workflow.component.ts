import {Component, ViewChild} from '@angular/core';
import {Project} from '../../../model/project.model';
import {ActivatedRoute, Router} from '@angular/router';
import {Subscription} from 'rxjs/Subscription';
import {Workflow, WorkflowNode, WorkflowNodeJoin, WorkflowNodeJoinTrigger, WorkflowNodeTrigger} from '../../../model/workflow.model';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {WorkflowTriggerComponent} from '../../../shared/workflow/trigger/workflow.trigger.component';
import {SemanticModalComponent} from 'ng-semantic';
import {TranslateService} from 'ng2-translate';
import {ToastService} from '../../../shared/toast/ToastService';
import {cloneDeep} from 'lodash';
import {WorkflowTriggerJoinComponent} from '../../../shared/workflow/join/trigger/trigger.join.component';
import {WorkflowJoinTriggerSrcComponent} from '../../../shared/workflow/join/trigger/src/trigger.src.component';
import {WorkflowGraphComponent} from '../graph/workflow.graph.component';
import {WorkflowRunService} from '../../../service/workflow/run/workflow.run.service';

declare var _: any;
@Component({
    selector: 'app-workflow',
    templateUrl: './workflow.html',
    styleUrls: ['./workflow.scss']
})
@AutoUnsubscribe()
export class WorkflowShowComponent {

    project: Project;
    detailedWorkflow: Workflow;
    workflowSubscription: Subscription;

    @ViewChild('workflowGraph')
    workflowGraph: WorkflowGraphComponent;
    @ViewChild('editTriggerComponent')
    editTriggerComponent: WorkflowTriggerComponent;
    @ViewChild('editJoinTriggerComponent')
    editJoinTriggerComponent: WorkflowTriggerJoinComponent;
    @ViewChild('workflowJoinTriggerSrc')
    workflowJoinTriggerSrc: WorkflowJoinTriggerSrcComponent;

    selectedNode: WorkflowNode;
    selectedTrigger: WorkflowNodeTrigger;
    selectedJoin: WorkflowNodeJoin;
    selectedJoinTrigger: WorkflowNodeJoinTrigger;

    loading = false;

    constructor(private activatedRoute: ActivatedRoute, private _workflowStore: WorkflowStore, private _router: Router,
                private _translate: TranslateService, private _toast: ToastService, private _workflowRun: WorkflowRunService) {
        // Update data if route change
        this.activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this.activatedRoute.params.subscribe(params => {
            let workflowName = params['workflowName'];
            if (this.project.key && workflowName) {
                if (this.workflowSubscription) {
                    this.workflowSubscription.unsubscribe();
                }

                if (!this.detailedWorkflow) {
                    this.workflowSubscription = this._workflowStore.getWorkflows(this.project.key, workflowName).subscribe(ws => {
                        if (ws) {
                            let updatedWorkflow = ws.get(this.project.key + '-' + workflowName);
                            if (updatedWorkflow && !updatedWorkflow.externalChange) {
                                this.detailedWorkflow = updatedWorkflow;
                            }
                        }
                    }, () => {
                        this._router.navigate(['/project', this.project.key]);
                    });
                }
            }
        });
    }

    public openDeleteJoinSrcModal(data: { source, target }) {
        let pID = Number(data.source.replace('node-', ''));
        let cID = Number(data.target.replace('join-', ''));

        this.selectedNode = Workflow.getNodeByID(pID, this.detailedWorkflow);
        this.selectedJoin = this.detailedWorkflow.joins.find(j => j.id === cID);

        if (this.workflowJoinTriggerSrc) {
            this.workflowJoinTriggerSrc.show({observable: true, closable: false, autofocus: false});
        }
    }

    public openEditTriggerModal(data: { source, target }) {
        let pID = Number(data.source.replace('node-', ''));
        let cID = Number(data.target.replace('node-', ''));
        let node = Workflow.getNodeByID(pID, this.detailedWorkflow);
        if (node && node.triggers) {
            for (let i = 0; i < node.triggers.length; i++) {
                if (node.triggers[i].workflow_dest_node_id === cID) {
                    this.selectedNode = cloneDeep(node);
                    this.selectedTrigger = cloneDeep(node.triggers[i]);
                    break;
                }
            }
        }
        if (this.editTriggerComponent) {
            setTimeout(() => {
                this.editTriggerComponent.show({observable: true, closable: false, autofocus: false});
            }, 1);

        }
    }

    public openEditJoinTriggerModal(data: { source, target }) {
        let pID = Number(data.source.replace('join-', ''));
        let cID = Number(data.target.replace('node-', ''));
        let join = this.detailedWorkflow.joins.find(j => j.id === pID);
        if (join && join.triggers) {
            this.selectedJoin = join;
            this.selectedJoinTrigger = cloneDeep(join.triggers.find(t => t.workflow_dest_node_id === cID));
        }
        if (this.editJoinTriggerComponent) {
            setTimeout(() => {
                this.editJoinTriggerComponent.show({observable: true, closable: false, autofocus: false});
            }, 1);

        }
    }

    public addSourceToJoin(data: { source: WorkflowNode, target: WorkflowNodeJoin }): void {
        let clonedWorkflow: Workflow = cloneDeep(this.detailedWorkflow);
        let currentJoin = clonedWorkflow.joins.find(j => j.id === data.target.id);
        this.selectedNode = data.source;
        if (currentJoin.source_node_id.find(id => id === this.selectedNode.id)) {
            return;
        }
        currentJoin.source_node_ref.push(this.selectedNode.ref);
        this.updateWorkflow(clonedWorkflow);
    }

    deleteJoinSrc(action: string): void {
        let clonedWorkflow: Workflow = cloneDeep(this.detailedWorkflow);

        switch (action) {
            case 'delete_join':
                clonedWorkflow.joins = clonedWorkflow.joins.filter(j => j.id !== this.selectedJoin.id);
                Workflow.removeOldRef(clonedWorkflow);
                break;
            default:
                let currentJoin = clonedWorkflow.joins.find(j => j.id === this.selectedJoin.id);
                currentJoin.source_node_ref = currentJoin.source_node_ref.filter(ref => ref !== this.selectedNode.ref);
        }

        this.updateWorkflow(clonedWorkflow, this.workflowJoinTriggerSrc.modal);
    }

    updateTrigger(): void {
        let clonedWorkflow: Workflow = cloneDeep(this.detailedWorkflow);
        let currentNode: WorkflowNode;
        if (clonedWorkflow.root.id === this.selectedNode.id) {
            currentNode = clonedWorkflow.root;
        } else if (clonedWorkflow.root.triggers) {
            currentNode = Workflow.getNodeByID(this.selectedNode.id, clonedWorkflow);
        }

        if (!currentNode) {
            return;
        }

        let trigToUpdate = currentNode.triggers.find(trig => trig.id === this.selectedTrigger.id);
        trigToUpdate.conditions = this.selectedTrigger.conditions;
        this.updateWorkflow(clonedWorkflow, this.editTriggerComponent.modal);
    }

    updateJoinTrigger(): void {
        let clonedWorkflow: Workflow = cloneDeep(this.detailedWorkflow);
        let currentJoin = clonedWorkflow.joins.find(j => j.id === this.selectedJoin.id);

        let trigToUpdate = currentJoin.triggers.find(trig => trig.id === this.selectedJoinTrigger.id);
        trigToUpdate.conditions = this.selectedJoinTrigger.conditions;
        this.updateWorkflow(clonedWorkflow, this.editJoinTriggerComponent.modal);
    }

    updateWorkflow(w: Workflow, modal?: SemanticModalComponent): void {
        this._workflowStore.updateWorkflow(this.project.key, w).first().subscribe(() => {
            this._toast.success('', this._translate.instant('workflow_updated'));
            if (modal) {
                modal.hide();
            }
            if (this.workflowGraph) {
                this.workflowGraph.toggleLinkJoin(false);
            }
        });
    }

    runWorkflow(): void {
        this.loading = true;
        this._workflowRun.runWorkflow(this.project.key, this.detailedWorkflow, {}).first().subscribe(wr => {
            this.loading = false;
            this._router.navigate(['/project', this.project.key, 'workflow', this.detailedWorkflow.name, 'run', wr.num]);
        }, () => {
            this.loading = false;
        });
    }
}
