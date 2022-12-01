import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component, Input,
    OnDestroy,
} from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { MenuItem, FlatNodeItem, TreeEvent } from 'app/shared/tree/tree.component';
import { ProjectService } from 'app/service/project/project.service';
import { Project, VCSProject } from 'app/model/project.model';
import { Observable, of } from 'rxjs';
import { map } from 'rxjs/operators';

@Component({
    selector: 'app-projectv2-sidebar',
    templateUrl: './project.sidebar.html',
    styleUrls: ['./project.sidebar.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2SidebarComponent implements OnDestroy {
    _currentProject: Project;
    get project(): Project {
        return this._currentProject;
    }
    @Input() set project(data: Project) {
        this._currentProject = data;
        if (data) {
            this.init();
        }
    }

    loading: boolean = true;
    currentWorkspace: FlatNodeItem[];
    currentIntegrations: FlatNodeItem[];
    panels: boolean[] = [true, false]

    ngOnDestroy(): void {
    }

    constructor(private _cd: ChangeDetectorRef, private _projectService: ProjectService) {}

    init(): void {
        this.currentWorkspace = [];
        this.loading = true;
        this._projectService.getVCSProject(this._currentProject.key).subscribe(vcsProjects => {
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
                this.loading = false;
                this._cd.markForCheck();
            }
        });
        this._cd.markForCheck();
    }

    loadRepositories(key: string, vcs: string): Observable<Array<FlatNodeItem>> {
        return this._projectService.getVCSRepositories(key, vcs).pipe(map((repos) => {
            return repos.map(r => {
                return <FlatNodeItem>{name: r.name, id: r.id, type: 'repository', expandable: true, level: 1,
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
                    existingEntities.push(<FlatNodeItem>{name: e.name, id: e.id, type: e.type, expandable: false, level: 3, icon: 'file', iconTheme: 'outline'})
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
        menu.push(<MenuItem>{ name: 'Add a repository', route: ['/', 'projectv2', 'vcs', vcs.name]});
        return menu;
    }

    handleWorkspaceEvent(e: TreeEvent): void {
        // TODO manage click on node title
        console.log(e);
    }

    togglePanel(i: number): void {
        this.panels[i] = !this.panels[i];
        this.panels = Object.assign([], this.panels)
        this._cd.markForCheck();
    }
}
