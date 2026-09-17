import { ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, Input, OnDestroy, OnInit } from '@angular/core';
import { NavigationEnd, Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { NzDrawerService } from 'ng-zorro-antd/drawer';
import { NzMessageService } from 'ng-zorro-antd/message';
import { filter, lastValueFrom, Subscription } from 'rxjs';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ErrorUtils } from 'app/shared/error.utils';
import { Project, ProjectRepository } from 'app/model/project.model';
import { VCSProject } from 'app/model/vcs.model';
import { EventV2Type, FullEventV2 } from 'app/model/event-v2.model';
import { ProjectService } from 'app/service/project/project.service';
import { RouterService } from 'app/service/services.module';
import { EventV2State } from 'app/store/event-v2.state';
import { PreferencesState } from 'app/store/preferences.state';
import * as actionPreferences from 'app/store/preferences.action';
import { ProjectV2RepositoryAddComponent, ProjectV2RepositoryAddComponentParams } from './repository-add/repository-add.component';

/**
 * The workspace tree: the vcs of the project and their repositories, declared or only listened to.
 * Opening a repository is the whole job; its content lives in the repository page.
 */
@Component({
    standalone: false,
    selector: 'app-projectv2-explore-sidebar',
    templateUrl: './explore-sidebar.html',
    styleUrls: ['./explore-sidebar.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2ExploreSidebarComponent implements OnInit, OnDestroy {
    /**
     * The tree of the last project explored. The view is rebuilt every time it is opened and the
     * repositories come from the vcs provider: coming back draws the last tree at once, then
     * refreshes it. Kept on the class, since it must outlive the component.
     */
    private static snapshot: {
        projectKey: string;
        vcss: Array<VCSProject>;
        repositories: { [vcs: string]: Array<ProjectRepository> };
    };

    @Input() project: Project;

    loading: boolean = true;
    vcss: Array<VCSProject> = [];
    repositories: { [vcs: string]: Array<ProjectRepository> } = {};
    expanded: { [vcs: string]: boolean } = {};

    routerSub: Subscription;
    eventV2Subscription: Subscription;

    private _cd = inject(ChangeDetectorRef);
    private _projectService = inject(ProjectService);
    private _messageService = inject(NzMessageService);
    private _store = inject(Store);
    private _router = inject(Router);
    private _routerService = inject(RouterService);
    private _drawerService = inject(NzDrawerService);

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.expanded = this._store.selectSnapshot(PreferencesState.selectProjectTreeExpandState(this.project.key));

        const snapshot = ProjectV2ExploreSidebarComponent.snapshot;
        if (snapshot && snapshot.projectKey === this.project.key) {
            this.vcss = snapshot.vcss;
            this.repositories = snapshot.repositories;
        }

        this.load();

        this.routerSub = this._router.events.pipe(filter(e => e instanceof NavigationEnd)).subscribe(() => {
            this.expandToRoute();
            this._cd.markForCheck();
        });
        this.eventV2Subscription = this._store.select(EventV2State.last).subscribe(event => this.handleEvent(event));
    }

    async load() {
        this.loading = true;
        this._cd.markForCheck();

        try {
            this.vcss = await lastValueFrom(this._projectService.listVCSProject(this.project.key));
            // A vcs is open unless the user closed it
            this.vcss.forEach(vcs => {
                if (!(vcs.name in this.expanded)) {
                    this.expanded[vcs.name] = true;
                }
            });
            this._cd.markForCheck();

            const [declared, distant] = await Promise.all([
                Promise.all(this.vcss.map(vcs => lastValueFrom(this._projectService.getVCSRepositories(this.project.key, vcs.name)))),
                lastValueFrom(this._projectService.getDistantRepositories(this.project.key))
            ]);
            this.repositories = {};
            this.vcss.forEach((vcs, i) => {
                // Repositories the project only listens to take place among the declared ones
                const listened = distant.filter(d => d.vcs_name === vcs.name).map(d => <ProjectRepository>{ name: d.repository, distant: true });
                this.repositories[vcs.name] = this.sortRepositories(declared[i].concat(listened));
            });
            this.expandToRoute();

            ProjectV2ExploreSidebarComponent.snapshot = {
                projectKey: this.project.key,
                vcss: this.vcss,
                repositories: this.repositories
            };
        } catch (e: any) {
            this._messageService.error(`Unable to load vcs and repositories: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }

        this.loading = false;
        this._cd.markForCheck();
    }

    clickVCS(vcs: VCSProject): void {
        this.expanded[vcs.name] = !this.expanded[vcs.name];
        this.saveExpandState();
        this._cd.markForCheck();
    }

    /** Closes every vcs; the page of a repository, now hidden, gives way to the overview. */
    collapseAll(): void {
        this.vcss.forEach(vcs => this.expanded[vcs.name] = false);
        this.saveExpandState();
        if (this.routeParams()['repoName']) {
            this._router.navigate(['/project', this.project.key, 'explore']);
        }
        this._cd.markForCheck();
    }

    openRepositoryAddDrawer(vcs: string): void {
        this._drawerService.create<ProjectV2RepositoryAddComponent, { params: ProjectV2RepositoryAddComponentParams }, string>({
            nzTitle: 'Add a new Repository',
            nzContent: ProjectV2RepositoryAddComponent,
            nzContentParams: {
                params: <ProjectV2RepositoryAddComponentParams>{ vcs }
            },
            nzSize: 'large'
        });
    }

    /** The vcs of the repository the url names is shown open, without touching the preference. */
    private expandToRoute(): void {
        const vcsName = this.routeParams()['vcsName'];
        if (vcsName && this.vcss.some(vcs => vcs.name === vcsName)) {
            this.expanded[vcsName] = true;
        }
    }

    /** Open is the default: only the vcs the user closed are remembered. */
    private saveExpandState(): void {
        const state: { [key: string]: boolean } = {};
        this.vcss.filter(vcs => this.expanded[vcs.name] === false).forEach(vcs => state[vcs.name] = false);
        this._store.dispatch(new actionPreferences.SaveProjectTreeExpandState({ projectKey: this.project.key, state }));
    }

    private async handleEvent(event: FullEventV2) {
        if (!event || [EventV2Type.EventRepositoryCreated, EventV2Type.EventRepositoryDeleted].indexOf(event.type) === -1) {
            return;
        }
        if (!this.repositories[event.vcs_name]) {
            return;
        }

        const others = this.repositories[event.vcs_name].filter(r => r.name !== event.repository);
        if (event.type === EventV2Type.EventRepositoryDeleted) {
            this.repositories[event.vcs_name] = others;
            // The page of a repository that no longer exists gives way to the overview
            const params = this.routeParams();
            if (params['vcsName'] === event.vcs_name && params['repoName'] === event.repository) {
                this._router.navigate(['/project', this.project.key, 'explore']);
            }
            this._cd.markForCheck();
            return;
        }

        // A repository the project only listened to becomes a declared one once added
        const repository = await lastValueFrom(this._projectService.getVCSRepository(this.project.key, event.vcs_name, event.repository));
        this.repositories[event.vcs_name] = this.sortRepositories(others.concat(repository));
        this.expanded[event.vcs_name] = true;
        this._cd.markForCheck();
    }

    private routeParams(): { [key: string]: string } {
        return this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root);
    }

    private sortRepositories(repositories: Array<ProjectRepository>): Array<ProjectRepository> {
        return repositories.sort((a, b) => a.name < b.name ? -1 : 1);
    }
}
