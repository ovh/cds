import {
    AfterViewInit,
    ChangeDetectorRef,
    Component,
    ComponentFactoryResolver,
    ComponentRef, HostListener,
    ViewChild,
    ViewContainerRef
} from '@angular/core';
import * as d3 from 'd3';
import * as dagreD3 from 'dagre-d3';
import {Project} from '../../../model/project.model';
import {ActivatedRoute, Router} from '@angular/router';
import {Subscription} from 'rxjs/Subscription';
import {Workflow, WorkflowNode, WorkflowNodeJoin, WorkflowNodeJoinTrigger, WorkflowNodeTrigger} from '../../../model/workflow.model';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {WorkflowNodeComponent} from '../../../shared/workflow/node/workflow.node.component';
import {WorkflowTriggerComponent} from '../../../shared/workflow/trigger/workflow.trigger.component';
import {SemanticModalComponent} from 'ng-semantic';
import {TranslateService} from 'ng2-translate';
import {ToastService} from '../../../shared/toast/ToastService';
import {WorkflowJoinComponent} from '../../../shared/workflow/join/workflow.join.component';
import {cloneDeep} from 'lodash';
import {WorkflowTriggerJoinComponent} from '../../../shared/workflow/join/trigger/trigger.join.component';
import {WorkflowJoinTriggerSrcComponent} from '../../../shared/workflow/join/trigger/src/trigger.src.component';

declare var _: any;
@Component({
    selector: 'app-workflow',
    templateUrl: './workflow.html',
    styleUrls: ['./workflow.scss'],
    entryComponents: [
        WorkflowNodeComponent,
        WorkflowJoinComponent
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
    selectedJoin: WorkflowNodeJoin;
    selectedJoinTrigger: WorkflowNodeJoinTrigger;
    linkWithJoin = false;

    nodesComponent = new Array<ComponentRef<WorkflowNodeComponent>>();
    joinsComponent = new Array<ComponentRef<WorkflowJoinComponent>>();

    // workflow graph
    @ViewChild('svgGraph', {read: ViewContainerRef}) svgContainer;
    g: dagreD3.graphlib.Graph;
    render = new dagreD3.render();
    svgWidth = 1500;


    @ViewChild('editTriggerComponent')
    editTriggerComponent: WorkflowTriggerComponent;
    @ViewChild('editJoinTriggerComponent')
    editJoinTriggerComponent: WorkflowTriggerJoinComponent;
    @ViewChild('workflowJoinTriggerSrc')
    workflowJoinTriggerSrc: WorkflowJoinTriggerSrcComponent;


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

    @HostListener('window:resize', ['$event'])
    onResize(event) {
        if (event) {
            this.svgWidth = event.target.innerWidth - 200;
        } else {
            console.log(window);
            this.svgWidth = window.innerWidth - 200;
        }

        let svg = d3.select('svg');
        let inner = d3.select('svg g');
        let svgWidth = +svg.attr('width');
        let xCenterOffset = (svgWidth - this.g.graph().width) / 2;
        inner.attr('transform', 'translate(' + xCenterOffset + ', 20)');
        svg.attr('height', this.g.graph().height + 40);
    }

    initWorkflow() {
        this.svgWidth = window.innerWidth - 200;
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
        svg.attr('width', this.svgWidth);

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

        // Add our custom arrow (a hollow-point)
        this.render.arrows()['customArraow'] = (parent, id, edge, type) => {
            let marker = parent.append('marker')
                .attr('id', id)
                .attr('viewBox', '0 0 10 10')
                .attr('refX', 7)
                .attr('refY', 5)
                .attr('markerUnits', 'strokeWidth')
                .attr('markerWidth', 4)
                .attr('markerHeight', 3)
                .attr('orient', 'auto');

            let path = marker.append('path')
                .attr('d', 'M 0 0 L 10 5 L 0 10 z')
                .style('stroke-width', 1)
                .style('stroke-dasharray', '1,0');
            dagreD3['util'].applyStyle(path, edge[type + 'Style']);
        };

        // Run the renderer. This is what draws the final graph.
        this.render(inner, this.g);

        // Center the graph
        this.onResize(null);

        setTimeout(() => {
            svg.selectAll('g.edgePath').on('click', d => {
                // Trigger between node and node
                if (d.v.indexOf('node-') === 0 && d.w.indexOf('node-') === 0) {
                    this.openEditTriggerModal(d.v, d.w);
                }
                // Join Trigger
                if (d.v.indexOf('join-') === 0) {
                    this.openEditJoinTriggerModal(d.v, d.w);
                }

                // Node Join Src
                if (d.v.indexOf('node-') === 0 && d.w.indexOf('join-') === 0) {
                    this.openDeleteJoinSrcModal(d.v, d.w);
                }
            });
        }, 1);
    }

    createEdge(from: string, to: string, options: {}): void {
        options['arrowhead'] = 'customArraow';
        this.g.setEdge(from, to, options);
    }

    createJoin(join: WorkflowNodeJoin): void {

        let nodeComponentFactory = this.componentFactoryResolver.resolveComponentFactory(WorkflowJoinComponent);
        let componentRef = nodeComponentFactory.create(this.svgContainer.parentInjector);

        componentRef.instance.workflow = this.detailedWorkflow;
        componentRef.instance.join = join;
        componentRef.instance.project = this.project;
        componentRef.instance.disabled = this.linkWithJoin;

        componentRef.instance.selectEvent.subscribe(j => {
            if (this.linkWithJoin && this.selectedNode) {
                this.addSourceToJoin(j);
            }
        });

        this.joinsComponent.push(componentRef);
        this.svgContainer.insert(componentRef.hostView);

        this.g.setNode('join-' + join.id, {
            shape: 'circle',
            label: () => {
                return componentRef.location.nativeElement;
            },
            class: 'join'
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
        componentRef.instance.disabled = this.linkWithJoin;
        this.nodesComponent.push(componentRef);
        componentRef.instance.linkJoinEvent.subscribe(n => {
            this.selectedNode = n;
            this.toggleLinkJoin(true);

        });

        return componentRef;
    }

    private openDeleteJoinSrcModal(parentID: string, childID: string) {
        if (this.linkWithJoin) {
            return;
        }
        let pID = Number(parentID.replace('node-', ''));
        let cID = Number(childID.replace('join-', ''));

        this.selectedNode = Workflow.getNodeByID(pID, this.detailedWorkflow);
        this.selectedJoin = this.detailedWorkflow.joins.find(j => j.id === cID);

        if (this.workflowJoinTriggerSrc) {
            this.workflowJoinTriggerSrc.show({observable: true, closable: false, autofocus: false});
        }
    }

    private openEditTriggerModal(parentID: string, childID: string) {
        if (this.linkWithJoin) {
            return;
        }
        let pID = Number(parentID.replace('node-', ''));
        let cID = Number(childID.replace('node-', ''));
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

    private openEditJoinTriggerModal(parentID: string, childID: string) {
        if (this.linkWithJoin) {
            return;
        }
        let pID = Number(parentID.replace('join-', ''));
        let cID = Number(childID.replace('node-', ''));
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

    addSourceToJoin(join: WorkflowNodeJoin): void {
        let clonedWorkflow: Workflow = cloneDeep(this.detailedWorkflow);
        let currentJoin = clonedWorkflow.joins.find(j => j.id === join.id);
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
                clonedWorkflow.joins = clonedWorkflow.joins.filter(j => j.id !==  this.selectedJoin.id);
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
            this.toggleLinkJoin(false);
        });
    }

    toggleLinkJoin(b: boolean): void {
        this.linkWithJoin = b;
        this.nodesComponent.forEach(c => {
            (<WorkflowNodeComponent>c.instance).disabled = this.linkWithJoin;
        });
        this.joinsComponent.forEach(c => {
            (<WorkflowJoinComponent>c.instance).disabled = this.linkWithJoin;
        });
    }
}
