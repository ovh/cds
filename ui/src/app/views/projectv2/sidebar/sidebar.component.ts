import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component, Input,
    OnDestroy, OnInit, ViewChild,
} from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { MenuItem, FlatNodeItem, TreeEvent, SelectedItem, TreeComponent } from 'app/shared/tree/tree.component';
import { ProjectService } from 'app/service/project/project.service';
import { Project, VCSProject } from 'app/model/project.model';
import { Observable, of, Subscription } from 'rxjs';
import { finalize, map } from 'rxjs/operators';
import { ActivatedRoute, Router } from '@angular/router';
import { SidebarEvent, SidebarService } from 'app/service/sidebar/sidebar.service';

@Component({
    selector: 'app-projectv2-sidebar',
    templateUrl: './project.sidebar.html',
    styleUrls: ['./project.sidebar.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2SidebarComponent implements OnDestroy, OnInit {
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
    currentIntegrations: FlatNodeItem[];
    panels: boolean[] = [true, false];

    sidebarServiceSub: Subscription;

    ngOnDestroy(): void {}

    constructor(private _cd: ChangeDetectorRef, private _projectService: ProjectService, private _router: Router, private _sidebarService: SidebarService,
                private _activatedRoute: ActivatedRoute) {
    }

    ngOnInit(): void {
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
            }
            this._cd.markForCheck();
        });
    }

    selectRepository(e: SidebarEvent): void {
        let si = <SelectedItem>{id: e.parent.id, type: 'vcs', child: {id: e.nodeID, name: e.nodeName, action: 'select', type: 'repository'}}
        this.tree.selectNode(si)
    }

    removeRepository(e: SidebarEvent): void {
        if (this.tree) {
            this.tree.removeNode(e.nodeID)
        }
    }

    loadWorkspace(si?: SelectedItem): void {
        this.currentWorkspace = [];
        this.refreshWorkspace = true;
        this.loading = true;
        this._projectService.listVCSProject(this._currentProject.key)
            .pipe(finalize(() => {
                this.loading = false;
                this.refreshWorkspace = false;
                if (this.tree && si) {
                    setTimeout(() => {
                        this.tree.selectNode(si);
                    }, 500);
                }
                this._cd.markForCheck();
            }))
            .subscribe(vcsProjects => {
            if (vcsProjects) {
                this.currentWorkspace = [];
                vcsProjects.forEach(vcs => {
                    let nodeItem = <FlatNodeItem>{name: vcs.name, id: vcs.id, type: 'vcs', expandable: true, level: 0, menu: this.getVCSMenu(vcs),
                        loadChildren: () => {
                            return this.loadRepositories(this._currentProject.key, vcs.name);
                        }
                    }
                    this.currentWorkspace.push(nodeItem);
                });
                this._cd.markForCheck();
            }
        });
        this._cd.markForCheck();
    }

    loadRepositories(key: string, vcs: string): Observable<Array<FlatNodeItem>> {
        return this._projectService.getVCSRepositories(key, vcs).pipe(map((repos) => {
            return repos.map(r => {
                return <FlatNodeItem>{name: r.name, parentName: vcs, id: r.id, type: 'repository', expandable: true, level: 1,
                    loadChildren: () => {
                        return this.loadEntities(this._currentProject.key, vcs, r.name);
                    }}
            })
        }));
    }

    loadEntities(key: string, vcs: string, repo: string): Observable<Array<FlatNodeItem>> {
        return this._projectService.getRepoEntities(key, vcs, repo).pipe(map((entities) => {
            let result = new Array<FlatNodeItem>();
            if (entities) {
                let m = new Map<string, FlatNodeItem[]>();
                entities.forEach(e => {
                    let existingEntities = m.get(e.type);
                    if (!existingEntities) {
                        existingEntities = [];
                    }
                    existingEntities.push(<FlatNodeItem>{name: e.name, parentName: repo, id: e.id, type: e.type, expandable: false, level: 3, icon: 'file', iconTheme: 'outline'})
                    m.set(e.type, existingEntities);
                });

                Array.from(m.keys()).forEach(k => {
                    let folderNode = <FlatNodeItem>{name: k, type: 'folder', expandable: true, level: 2, id: k, loading: false,
                        loadChildren: () => {
                            return of(m.get(k));
                        }}
                    result.push(folderNode);
                })
            }
            return result;
        }));
    }

    getVCSMenu(vcs: VCSProject): MenuItem[] {
        let menu = [];
        menu.push(<MenuItem>{ name: 'Add a repository', route: ['/', 'projectv2', this.project.key, 'vcs', vcs.name, 'repository']});
        return menu;
    }

    handleWorkspaceEvent(e: TreeEvent): void {
        switch (e.node.type) {
            case 'vcs':
                // TODO go to vcs view
                break;
            case 'repository':
                if (e.eventType === 'select') {
                    this._router.navigate(['/', 'projectv2', this.project.key, 'vcs', e.node.parentName, 'repository', e.node.name]).then();
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
