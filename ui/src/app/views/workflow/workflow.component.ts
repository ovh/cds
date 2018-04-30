import {Component, ViewChild, OnInit} from '@angular/core';
import {SemanticSidebarComponent} from 'ng-semantic/ng-semantic';
import {ActivatedRoute, NavigationStart, Router} from '@angular/router';
import {Project} from '../../model/project.model';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../shared/decorator/autoUnsubscribe';
import {Workflow} from '../../model/workflow.model';
import {WorkflowStore} from '../../service/workflow/workflow.store';
import {ProjectStore} from '../../service/project/project.store';
import {WorkflowCoreService} from '../../service/workflow/workflow.core.service';
import {ToastService} from '../../shared/toast/ToastService';
import {TranslateService} from '@ngx-translate/core';
import {finalize} from 'rxjs/operators';
import {WorkflowSidebarMode, WorkflowSidebarStore} from '../../service/workflow/workflow.sidebar.store';
import {WorkflowRunService} from '../../service/workflow/run/workflow.run.service';
import {WorkflowEventStore} from '../../service/workflow/workflow.event.store';
import {EventStore} from '../../service/event/event.store';
import {EventSubscription} from '../../model/event.model';

@Component({
    selector: 'app-workflow',
    templateUrl: './workflow.html',
    styleUrls: ['./workflow.scss']
})
@AutoUnsubscribe()
export class WorkflowComponent implements OnInit {

    project: Project;
    workflow: Workflow;
    workflowSubscription: Subscription;
    projectSubscription: Subscription;

    loading = true;
    loadingFav = false;

    // Sidebar data
    sideBarModeSubscription: Subscription;
    sidebarMode = WorkflowSidebarMode.RUNS;
    sidebarModes = WorkflowSidebarMode;

    asCodeEditorSubscription: Subscription;
    asCodeEditorOpen: boolean;

    // Selected node
    selectedNodeID: number;

    @ViewChild('invertedSidebar')
    sidebar: SemanticSidebarComponent;

    constructor(private _activatedRoute: ActivatedRoute,
                private _workflowStore: WorkflowStore,
                private _workflowRunService: WorkflowRunService,
                private _workflowEventStore: WorkflowEventStore,
                private _router: Router,
                private _projectStore: ProjectStore,
                public _sidebarStore: WorkflowSidebarStore,
                private _workflowCore: WorkflowCoreService,
                private _toast: ToastService,
                private _translate: TranslateService,
                private _eventStore: EventStore) {
        this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this.asCodeEditorSubscription = this._workflowCore.getAsCodeEditor()
          .subscribe((state) => {
              if (state != null) {
                  this.asCodeEditorOpen = state.open;
              }
          });

        this.initSidebar();

        // Workflow subscription
        this._activatedRoute.params.subscribe(p => {
            let workflowName = p['workflowName'];
            let key = p['key'];

            if (key && workflowName) {
                if (this.workflowSubscription) {
                    this.workflowSubscription.unsubscribe();
                }
                this.loading = true;
                this.workflowSubscription = this._workflowStore.getWorkflows(key, workflowName)
                    .subscribe(ws => {
                        if (ws) {
                            let updatedWorkflow = ws.get(key + '-' + workflowName);
                            if (updatedWorkflow && !updatedWorkflow.externalChange) {

                                if (!this.workflow || (this.workflow && updatedWorkflow.id !== this.workflow.id)) {
                                    this.initRuns(key, workflowName);
                                }
                                this.workflow = updatedWorkflow;
                                if (this.selectedNodeID) {
                                    this._workflowEventStore.setSelectedNode(Workflow.getNodeByID(this.selectedNodeID, this.workflow), true);
                                }
                            }
                        }
                        this.loading = false;
                    }, () => {
                        this.loading = false;
                        this._router.navigate(['/project', key]);
                    });
            }
        });

        this._activatedRoute.queryParams.subscribe(qps => {
            if (qps['node_id']) {
                this.selectedNodeID = Number(qps['node_id']);
                if (this.workflow) {
                    this._workflowEventStore.setSelectedNode(Workflow.getNodeByID(this.selectedNodeID, this.workflow), true);
                }
            }
        });

        // unselect all when returning on workflow main page
        this._router.events.subscribe(e => {
            if (e instanceof NavigationStart && this.workflow) {
                if (e.url.indexOf('/project/' + this.project.key + '/workflow/') === 0 && e.url.indexOf('/run/') === -1) {
                    this._workflowEventStore.unselectAll();
                }
            }
        });
    }

    initRuns(key: string, workflowName: string): void {
        this._workflowRunService.runs(key, workflowName, '50').subscribe(wrs => {
            this._workflowEventStore.pushWorkflowRuns(wrs);

            let f = new EventSubscription();
            f.key = key;
            f.workflow_name = workflowName;
            f.runs = true;
            this._eventStore.changeFilter(f);
        });
    }

    initSidebar(): void {
        // Mode of sidebar
        this.sideBarModeSubscription = this._sidebarStore.sidebarMode().subscribe(m => {
            this.sidebarMode = m;
        })
    }

    ngOnInit() {
        this.projectSubscription = this._projectStore.getProjects(this.project.key)
          .subscribe((proj) => {
            if (!this.project || !proj || !proj.get(this.project.key)) {
              return;
            }
            this.project = proj.get(this.project.key);
          });
    }

    updateFav() {
        if (this.loading) {
            return;
        }
        this.loadingFav = true;
        this._workflowStore.updateFavorite(this.project.key, this.workflow.name)
            .pipe(finalize(() => this.loadingFav = false))
            .subscribe(() => this._toast.success('', this._translate.instant('common_favorites_updated')))
    }

    changeToRunsMode(): void {
        this._workflowEventStore.setSelectedNode(null, false);
        this._workflowEventStore.setSelectedNodeRun(null);
        this._workflowEventStore.setSelectedHook(null);
        this._workflowEventStore.setSelectedJoin(null);
        this._sidebarStore.changeMode(WorkflowSidebarMode.RUNS);
    }
}
