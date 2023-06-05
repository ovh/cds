import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component, Input,
    OnDestroy, ViewChild,
} from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import {
    MenuItem,
    FlatNodeItem,
    TreeEvent,
    SelectedItem,
    TreeComponent,
    FlatNodeItemSelect
} from 'app/shared/tree/tree.component';
import { ProjectService } from 'app/service/project/project.service';
import { Project, VCSProject} from 'app/model/project.model';
import { Observable, of, Subscription } from 'rxjs';
import { map } from 'rxjs/operators';
import { Router } from '@angular/router';
import { SidebarEvent, SidebarService } from 'app/service/sidebar/sidebar.service';
import { AnalysisService } from "../../../service/analysis/analysis.service";
import {EntityAction, EntityWorkerModel, EntityWorkflow} from "../../../model/entity.model";

@Component({
    selector: 'app-projectv2-sidebar',
    templateUrl: './sidebar.html',
    styleUrls: ['./sidebar.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2SidebarComponent implements OnDestroy, AfterViewInit {
    _currentProject: Project;
    get project(): Project {
        return this._currentProject;
    }
    @Input() set project(data: Project) {
        this._currentProject = data;
        if (data) {
            this.loadWorkspace();
        }
    }

    @ViewChild('treeWorkspace') tree: TreeComponent

    loading: boolean = true;
    refreshWorkspace: boolean = false;
    currentWorkspace: FlatNodeItem[];
    panels: boolean[] = [true, false];

    sidebarServiceSub: Subscription;
    analysisServiceSub: Subscription;

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    constructor(
        private _cd: ChangeDetectorRef,
        private _projectService: ProjectService,
        private _router: Router,
        private _sidebarService: SidebarService,
        private _analysisService: AnalysisService
    ) { }

    ngAfterViewInit(): void {
        this.sidebarServiceSub = this._sidebarService.getObservable().subscribe(e => {
            switch (e?.nodeType) {
                case 'vcs':
                    // TODO select vcs
                    break;
                case 'repository':
                    switch (e.action) {
                        case 'remove':
                            this.removeRepository(e);
                            break;
                        case 'select':
                            this.selectRepository(e);
                            break;
                    }
                    break;
                case EntityWorkerModel:
                    switch (e.action) {
                        case 'select':
                            this.selectEntity(e);
                            break;
                    }
                    break;
            }
            this._cd.markForCheck();
        });

        this.analysisServiceSub = this._analysisService.getObservable().subscribe(e => {
            if (e) {
                this.tree.handleAnalysisEvent(e);
            }
        });
    }

    selectEntity(e: SidebarEvent): void {
        let si = <SelectedItem>{
            id: e.parentIDs[0],
            type: 'vcs',
            child: {
                id: e.parentIDs[1],
                type: 'repository',
                child: { id: EntityWorkerModel, type: 'folder', child: { id: e.nodeID, name: e.nodeName, action: 'select', type: EntityWorkerModel } }
            }
        }
        this.tree.selectNode(si);
    }

    selectRepository(e: SidebarEvent): void {
        let si = <SelectedItem>{
            id: e.parentIDs[0],
            type: 'vcs',
            child: { id: e.nodeID, name: e.nodeName, action: 'select', type: 'repository' }
        }
        this.tree.selectNode(si);
    }

    removeRepository(e: SidebarEvent): void {
        if (this.tree) {
            this.tree.removeNode(e.nodeID)
        }
    }

    clickRefresh(e: Event) {
        e.stopPropagation();
        this.loadWorkspace();
    }

    async loadWorkspace() {
        this.currentWorkspace = [];
        this.refreshWorkspace = true;
        this.loading = true;
        this._cd.markForCheck();

        const vcsProjects = await this._projectService.listVCSProject(this._currentProject.key).toPromise();
        if (vcsProjects) {
            this.currentWorkspace = vcsProjects.map(vcs => (<FlatNodeItem>{
                name: vcs.name,
                id: vcs.id,
                type: 'vcs',
                expandable: true,
                clickable: false,
                level: 0,
                menu: this.getVCSMenu(vcs),
                loadChildren: () => this.loadRepositories(this._currentProject.key, vcs.name)
            }));
        }

        this.loading = false;
        this.refreshWorkspace = false;
        this._cd.markForCheck();
    }

    loadRepositories(key: string, vcs: string): Observable<Array<FlatNodeItem>> {
        return this._projectService.getVCSRepositories(key, vcs).pipe(map((repos) => {
            return repos.map(r => {
                let nodeItem = <FlatNodeItem>{
                    name: r.name,
                    parentNames: [vcs],
                    id: r.id,
                    type: 'repository',
                    expandable: true,
                    clickable: true,
                    level: 1
                };
                nodeItem.loadChildren = () => {
                    const currentBranch = this._router?.routerState?.snapshot?.root?.queryParams['branch'];
                    return this.loadEntities(this._currentProject.key, vcs, r.name, currentBranch);
                };
                nodeItem.onOpen = () => {
                    const currentBranch = this._router?.routerState?.snapshot?.root?.queryParams['branch'];
                    return this._projectService.getVCSRepositoryBranches(key, vcs, r.name, 50).pipe(map(bs => {
                        nodeItem.select = <FlatNodeItemSelect>{ options: [] };
                        nodeItem.select.options = bs.map(b => {
                            if (b.display_id === currentBranch) {
                                nodeItem.select.selected = b.display_id;
                            }
                            if (b.default && !nodeItem.select.selected) {
                                nodeItem.select.selected = b.display_id;
                            }
                            return { key: b.display_id, value: b.display_id }
                        });
                        nodeItem.select.onchange = () => {
                            nodeItem.loadChildren = () => {
                                return this.loadEntities(this._currentProject.key, vcs, r.name, nodeItem.select.selected);
                            };
                            this.tree.resetChildren(nodeItem);
                            this._router.navigate([], { queryParams: { branch: nodeItem.select.selected } }).then();
                        }
                        this.tree.refresh();
                    }));
                }
                return nodeItem;
            })
        }));
    }

    loadEntities(key: string, vcs: string, repo: string, branch?: string): Observable<Array<FlatNodeItem>> {
        return this._projectService.getRepoEntities(key, vcs, repo, branch).pipe(map((entities) => {
            let result = new Array<FlatNodeItem>();
            if (entities) {
                let m = new Map<string, FlatNodeItem[]>();
                entities.forEach(e => {
                    let existingEntities = m.get(e.type);
                    if (!existingEntities) {
                        existingEntities = [];
                    }
                    existingEntities.push(<FlatNodeItem>{ name: e.name, parentNames: [vcs, repo], id: e.id, type: e.type, expandable: false, clickable: true, level: 3, icon: 'file', iconTheme: 'outline' })
                    m.set(e.type, existingEntities);
                });
                Array.from(m.keys()).forEach(k => {
                    result.push(<FlatNodeItem>{
                        name: k, type: 'folder', expandable: true, clickable: false, level: 2, id: k, loading: false,
                        loadChildren: () => of(m.get(k))
                    });
                })
            }
            return result;
        }));
    }

    getVCSMenu(vcs: VCSProject): MenuItem[] {
        return [<MenuItem>{
            name: 'Add a repository',
            route: ['/', 'projectv2', this.project.key, 'vcs', vcs.name, 'repository']
        }];
    }

    handleWorkspaceEvent(e: TreeEvent): void {
        switch (e.node.type) {
            case 'vcs':
                // TODO go to vcs view
                break;
            case 'repository':
                if (e.eventType === 'select') {
                    this._router.navigate(['/', 'projectv2', this.project.key, 'vcs', e.node.parentNames[0], 'repository', e.node.name], {
                        queryParamsHandling: 'preserve'
                    }).then();
                }
                break;
            case EntityWorkerModel:
                if (e.eventType === 'select') {
                    this._router.navigate(['/', 'projectv2', this.project.key, 'vcs', e.node.parentNames[0], 'repository', e.node.parentNames[1], 'workermodel', e.node.name], {
                        queryParamsHandling: 'preserve'
                    }).then();
                }
                break;
            case EntityAction:
                if (e.eventType === 'select') {
                    this._router.navigate(['/', 'projectv2', this.project.key, 'vcs', e.node.parentNames[0], 'repository', e.node.parentNames[1], 'action', e.node.name], {
                        queryParamsHandling: 'preserve'
                    }).then();
                }
                break;
            case EntityWorkflow:
                if (e.eventType === 'select') {
                    this._router.navigate(['/', 'projectv2', this.project.key, 'vcs', e.node.parentNames[0], 'repository', e.node.parentNames[1], 'workflow', e.node.name], {
                        queryParamsHandling: 'preserve'
                    }).then();
                }
                break;
        }
    }

    togglePanel(i: number): void {
        this.panels[i] = !this.panels[i];
        this.panels = Object.assign([], this.panels)
        this._cd.markForCheck();
    }
}
