import {Component, ViewChild} from '@angular/core';
import {SemanticSidebarComponent} from 'ng-semantic/ng-semantic';
import {ActivatedRoute, ResolveEnd, Router} from '@angular/router';
import {Project} from '../../model/project.model';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../shared/decorator/autoUnsubscribe';
import {Workflow} from '../../model/workflow.model';
import {WorkflowStore} from '../../service/workflow/workflow.store';
import {RouterService} from '../../service/router/router.service';
import {WorkflowCoreService} from '../../service/workflow/workflow.core.service';
import {finalize} from 'rxjs/operators';

@Component({
    selector: 'app-workflow',
    templateUrl: './workflow.html',
    styleUrls: ['./workflow.scss']
})
@AutoUnsubscribe()
export class WorkflowComponent {

    project: Project;
    workflow: Workflow;
    loading = true;
    number: number;
    workflowSubscription: Subscription;
    sidebarOpen: boolean;
    currentNodeName: string;

    @ViewChild('invertedSidebar')
    sidebar: SemanticSidebarComponent;

    constructor(private _activatedRoute: ActivatedRoute, private _workflowStore: WorkflowStore, private _router: Router,
                private _routerService: RouterService, private _workflowCore: WorkflowCoreService) {
        this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this._workflowCore.getSidebarStatus().subscribe(b => {
            this.sidebarOpen = b;
        });

        this._activatedRoute.params.subscribe(p => {
            let workflowName = p['workflowName'];
            if (this.project.key && workflowName) {
                if (this.workflowSubscription) {
                    this.workflowSubscription.unsubscribe();
                }

                this.workflowSubscription = this._workflowStore.getWorkflows(this.project.key, workflowName)
                    .subscribe(ws => {
                        if (ws) {
                            let updatedWorkflow = ws.get(this.project.key + '-' + workflowName);
                            if (updatedWorkflow && !updatedWorkflow.externalChange) {
                                this.workflow = updatedWorkflow;
                            }
                        }
                        this.loading = false;
                    }, () => {
                        this.loading = false;
                        this._router.navigate(['/project', this.project.key]);
                    });

            }

        });

        let snapshotparams = this._routerService.getRouteSnapshotParams({}, this._activatedRoute.snapshot);
        if (snapshotparams) {
            this.number = snapshotparams['number'];
        }
        let qp = this._routerService.getRouteSnapshotQueryParams({}, this._activatedRoute.snapshot);
        if (qp) {
            this.currentNodeName = qp['name'];
        }

        this._router.events.subscribe(p => {
            if (p instanceof ResolveEnd) {
                let params = this._routerService.getRouteSnapshotParams({}, p.state.root);
                let queryParams = this._routerService.getRouteSnapshotQueryParams({}, p.state.root);
                this.currentNodeName = queryParams['name'];
                this.number = params['number'];
            }
        });
    }

    toggleSidebar(): void {
        this._workflowCore.moveSideBar(!this.sidebarOpen);
    }
}
