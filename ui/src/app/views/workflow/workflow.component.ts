import {Component, EventEmitter, OnInit, QueryList, ViewChild, ViewChildren} from '@angular/core';
import { ActivatedRoute, NavigationStart, Params, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { SemanticSidebarComponent } from 'ng-semantic/ng-semantic';
import {SuiPopup, SuiPopupController, SuiPopupTemplateController} from 'ng2-semantic-ui/dist';
import { Subscription } from 'rxjs';
import { debounceTime, finalize } from 'rxjs/operators';
import { Project } from '../../model/project.model';
import { Workflow } from '../../model/workflow.model';
import { WorkflowRun } from '../../model/workflow.run.model';
import { ProjectStore } from '../../service/project/project.store';
import { RouterService } from '../../service/router/router.service';
import { WorkflowRunService } from '../../service/workflow/run/workflow.run.service';
import { WorkflowCoreService } from '../../service/workflow/workflow.core.service';
import { WorkflowEventStore } from '../../service/workflow/workflow.event.store';
import { WorkflowSidebarMode, WorkflowSidebarStore } from '../../service/workflow/workflow.sidebar.store';
import { WorkflowStore } from '../../service/workflow/workflow.store';
import { AutoUnsubscribe } from '../../shared/decorator/autoUnsubscribe';
import { ToastService } from '../../shared/toast/ToastService';
import { WorkflowTemplateModalComponent } from '../../shared/workflow-template/modal/workflow-template.modal.component';
import {WorkflowSaveAsCodeComponent} from '../../shared/workflow/modal/save-as-code/save.as.code.component';

@Component({
    selector: 'app-workflow',
    templateUrl: './workflow.html',
    styleUrls: ['./workflow.scss']
})
@AutoUnsubscribe(['onScroll'])
export class WorkflowComponent implements OnInit {
    @ViewChild('templateForm')
    templateFormComponent: WorkflowTemplateModalComponent;

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
    asCodeEditorOpen = false;

    @ViewChild('invertedSidebar')
    sidebar: SemanticSidebarComponent;
    @ViewChild('saveAsCode')
    saveAsCode: WorkflowSaveAsCodeComponent;
    @ViewChild('popup')
    popupFromlRepository: SuiPopup;
    @ViewChildren(SuiPopupController) popups: QueryList<SuiPopupController>;
    @ViewChildren(SuiPopupTemplateController) popups2: QueryList<SuiPopupTemplateController<SuiPopup>>;

    onScroll = new EventEmitter<boolean>();
    selectedNodeID: number;
    selectedNodeRef: string;
    selectecHookRef: string;

    runSubscription: Subscription;
    workflowRun: WorkflowRun;

    showButtons = false;
    loadingPopupButton = false;

    constructor(private _activatedRoute: ActivatedRoute,
        private _workflowStore: WorkflowStore,
        private _workflowRunService: WorkflowRunService,
        private _workflowEventStore: WorkflowEventStore,
        private _router: Router,
        private _routerService: RouterService,
        private _projectStore: ProjectStore,
        public _sidebarStore: WorkflowSidebarStore,
        private _workflowCore: WorkflowCoreService,
        private _toast: ToastService,
        private _translate: TranslateService) {
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

        this._activatedRoute.queryParams.subscribe(qps => {
            if (qps['node_id']) {
                this.selectedNodeID = Number(qps['node_id']);
                delete this.selectecHookRef;
            }
            if (qps['node_ref']) {
                this.selectedNodeRef = qps['node_ref'];
                delete this.selectecHookRef;
            }
            if (qps['hook_ref']) {
                this.selectecHookRef = qps['hook_ref'];
                delete this.selectedNodeRef;
                delete this.selectedNodeID;
            }
        });

        // Workflow subscription
        this._activatedRoute.params.subscribe(params => {
            let workflowName = params['workflowName'];
            let key = params['key'];

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

                                if (this.selectecHookRef) {
                                    let h = Workflow.getHookByRef(this.selectecHookRef, this.workflow);
                                    if (h) {
                                        this._workflowEventStore.setSelectedHook(h);
                                    }
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

        // unselect all when returning on workflow main page
        this._router.events.subscribe(e => {
            if (e instanceof NavigationStart && this.workflow) {
                if (e.url.indexOf('/project/' + this.project.key + '/workflow/') === 0 && e.url.indexOf('/run/') === -1) {
                    this._workflowEventStore.setSelectedRun(null);
                }
            }
        });

        this.runSubscription = this._workflowEventStore.selectedRun().subscribe(wr => {
            if (wr) {
                this.workflowRun = wr;
            } else {
                delete this.workflowRun;
            }
        });
    }

    initRuns(key: string, workflowName: string): void {
        this._workflowEventStore.setListingRuns(true);
        this._workflowRunService.runs(key, workflowName, '50')
            .subscribe(wrs => {
                this._workflowEventStore.setListingRuns(false);
                this._workflowEventStore.pushWorkflowRuns(wrs);
            });
    }

    initSidebar(): void {
        // Mode of sidebar
        this.sideBarModeSubscription = this._sidebarStore.sidebarMode()
            .pipe(debounceTime(150))
            .subscribe(m => setTimeout(() => this.sidebarMode = m, 0));
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
        let activatedRoute = this._routerService.getActivatedRoute(this._activatedRoute);
        let queryParams: Params;
        if (activatedRoute.snapshot.params['nodeId'] && activatedRoute.snapshot.queryParams['name']) {
            queryParams = {
                'name': activatedRoute.snapshot.queryParams['name'],
            };
        }

        this._router.navigate([], { relativeTo: activatedRoute, queryParams });
        if (!activatedRoute.snapshot.params['nodeId']) {
            this._workflowEventStore.setSelectedNode(null, true);
            this._workflowEventStore.setSelectedNodeRun(null, true);
        }
        this._sidebarStore.changeMode(WorkflowSidebarMode.RUNS);
    }

    showTemplateFrom(): void {
        if (this.templateFormComponent) {
            this.templateFormComponent.show();
        }
    }

    migrateAsCode(): void {
        this.loadingPopupButton = true;
        this._workflowStore.migrateAsCode(this.project.key, this.workflow.name)
            .pipe(finalize(() => this.loadingPopupButton = false ))
            .subscribe((ope) => {
            this.showButtons = false;
            this.popupFromlRepository.close();
            this.saveAsCode.show(ope);
        });
    }
}
