import {
    AfterViewInit,
    ChangeDetectorRef,
    Component,
    ComponentFactoryResolver,
    ComponentRef,
    ViewChild,
    ViewContainerRef
} from '@angular/core';
import * as d3 from 'd3';
import * as dagreD3 from 'dagre-d3';
import {Project} from '../../../model/project.model';
import {ActivatedRoute, Router} from '@angular/router';
import {Subscription} from 'rxjs/Subscription';
import {Workflow, WorkflowNode, WorkflowNodeJoin, WorkflowNodeTrigger} from '../../../model/workflow.model';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {WorkflowNodeComponent} from '../../../shared/workflow/node/workflow.node.component';
import {Pipeline} from '../../../model/pipeline.model';
import {WorkflowTriggerComponent} from '../../../shared/workflow/trigger/workflow.trigger.component';
import {SemanticModalComponent} from 'ng-semantic';
import {TranslateService} from 'ng2-translate';
import {ToastService} from '../../../shared/toast/ToastService';

declare var _: any;
@Component({
    selector: 'app-workflow',
    templateUrl: './workflow.html',
    styleUrls: ['./workflow.scss'],
    entryComponents: [
        WorkflowNodeComponent
    ]
})
@AutoUnsubscribe()
export class WorkflowShowComponent implements AfterViewInit {

    project: Project;
    detailedWorkflow: Workflow;
    workflowSubscription: Subscription;

    viewInit = false;

    selectedNode: WorkflowNode;
    selectedTrigger: WorkflowNodeTrigger;

    // workflow graph
    @ViewChild('svgGraph', {read: ViewContainerRef}) svgContainer;
    g: dagreD3.graphlib.Graph;
    render = new dagreD3.render();

    @ViewChild('editTriggerComponent')
    editTriggerComponent: WorkflowTriggerComponent;

    constructor(private activatedRoute: ActivatedRoute, private _workflowStore: WorkflowStore, private _router: Router,
                private componentFactoryResolver: ComponentFactoryResolver, private _cd: ChangeDetectorRef,
                private _translate: TranslateService, private _toast: ToastService) {
        // Update data if route change
        this.activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this.activatedRoute.params.subscribe(params => {
            let key = params['key'];
            let workflowName = params['workflowName'];
            if (key && workflowName) {
                if (this.workflowSubscription) {
                    this.workflowSubscription.unsubscribe();
                }

                if (!this.detailedWorkflow) {
                    this.workflowSubscription = this._workflowStore.getWorkflows(key, workflowName).subscribe(ws => {
                        if (ws) {
                            let updatedWorkflow = ws.get(key + '-' + workflowName);
                            if (updatedWorkflow && !updatedWorkflow.externalChange) {
                                this.detailedWorkflow = updatedWorkflow;
                                if (this.viewInit) {
                                    this.initWorkflow();
                                }
                            }
                        }
                    }, () => {
                        this._router.navigate(['/project', key]);
                    });
                }
            }
        });
    }

    showTab(tab: string): void {
        this._router.navigateByUrl('/project/' + this.project.key + '/workflow/' + this.detailedWorkflow.name + '?tab=' + tab);
    }

    ngAfterViewInit(): void {
        this.viewInit = true;
        if (this.detailedWorkflow) {
            this.initWorkflow();
        }

    }

    initWorkflow() {
        // this.g = new dagreD3.graphlib.Graph().setGraph({ directed: false, rankDir: 'LR'});
        this.g = new dagreD3.graphlib.Graph().setGraph({directed: false});
        if (this.detailedWorkflow.root) {
            this.createNode(this.detailedWorkflow.root);
        }
        if (this.detailedWorkflow.joins) {
            this.detailedWorkflow.joins.forEach(j => {
                this.createJoin(j);
            });

        }

        // Set up an SVG group so that we can translate the final graph.
        let svg = d3.select('svg');
        let inner = d3.select('svg g');
        /* FIXME : resize child
         let zoom = d3.behavior.zoom().on('zoom', () => {
         inner.attr('transform', 'translate(' + (<ZoomEvent>d3.event).translate + ')' + 'scale(' + (<ZoomEvent>d3.event).scale + ')');
         //this.centerGraph(svg, inner);
         });
         svg.call(zoom);
         */
        this.g.transition = (selection) => {
            return selection.transition().duration(500);
        };

        // Run the renderer. This is what draws the final graph.
        this.render(inner, this.g);

        // Center the graph
        this.centerGraph(svg, inner);

        setTimeout(() => {
            svg.selectAll('g.edgePath').on('click', d => {
                this.openEditTriggerModal(d.v, d.w);
            });
        }, 1);
    }

    centerGraph(svg: any, inner: any): void {
        let svgWidth = +svg.attr('width');
        let xCenterOffset = (svgWidth - this.g.graph().width) / 2;
        inner.attr('transform', 'translate(' + xCenterOffset + ', 20)');
        svg.attr('height', this.g.graph().height + 40);
    }

    createEdge(from: string, to: string, options: {}): void {
        this.g.setEdge(from, to, options);
    }

    createJoin(join: WorkflowNodeJoin): void {

        let nodeComponentFactory = this.componentFactoryResolver.resolveComponentFactory(WorkflowNodeComponent);
        let componentRef = nodeComponentFactory.create(this.svgContainer.parentInjector);

        // FIXME/ use a WorkflowJoinComponent
        let fake = new WorkflowNode();
        fake.pipeline = new Pipeline();
        fake.pipeline.name = 'JOINNNN';
        componentRef.instance.node = fake;
        this.svgContainer.insert(componentRef.hostView);

        this.g.setNode('join-' + join.id, {
            label: () => {
                return componentRef.location.nativeElement;
            }
        });

        if (join.source_node_id) {
            join.source_node_id.forEach(nodeID => {
                this.createEdge('node-' + nodeID, 'join-' + join.id, {});
            });
        }

        if (join.triggers) {
            join.triggers.forEach(t => {
                this.createNode(t.workflow_dest_node);
                this.createEdge('join-' + join.id, 'node-' + t.workflow_dest_node.id, {id: 'trigger-' + t.id});
            });
        }
    }

    createNode(node: WorkflowNode): void {
        let componentRef = this.createNodeComponent(node);
        this.svgContainer.insert(componentRef.hostView);
        this.g.setNode('node-' + node.id, {
            label: () => {
                return componentRef.location.nativeElement;
            }
        });
        if (node.triggers) {
            node.triggers.forEach(t => {
                this.createNode(t.workflow_dest_node);
                this.createEdge('node-' + node.id, 'node-' + t.workflow_dest_node.id, {id: 'trigger-' + t.id});
            });
        }
    }

    createNodeComponent(node: WorkflowNode): ComponentRef<WorkflowNodeComponent> {
        let nodeComponentFactory = this.componentFactoryResolver.resolveComponentFactory(WorkflowNodeComponent);
        let componentRef = nodeComponentFactory.create(this.svgContainer.parentInjector);
        componentRef.instance.node = node;
        componentRef.instance.workflow = this.detailedWorkflow;
        componentRef.instance.project = this.project;

        return componentRef;
    }

    private openEditTriggerModal(parentID: string, childID: string) {
        let pID = Number(parentID.replace('node-', ''));
        let cID = Number(childID.replace('node-', ''));
        let node = Workflow.getNodeByID(pID, this.detailedWorkflow);
        if (node && node.triggers) {
            for (var i=0; i<node.triggers.length; i++) {
                if (node.triggers[i].workflow_dest_node_id === cID) {
                    this.selectedNode = _.cloneDeep(node);
                    this.selectedTrigger = _.cloneDeep(node.triggers[i]);
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

    updateTrigger(): void {
        let clonedWorkflow: Workflow = _.cloneDeep(this.detailedWorkflow);
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

    updateWorkflow(w: Workflow, modal: SemanticModalComponent): void {
        this._workflowStore.updateWorkflow(this.project.key, w).first().subscribe(() => {
            this._toast.success('', this._translate.instant('workflow_updated'));
            modal.hide();
        });
    }
}
