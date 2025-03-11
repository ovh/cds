import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { NavigationEnd, Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { APIConfig } from 'app/model/config.service';
import { Help } from 'app/model/help.model';
import { Project } from 'app/model/project.model';
import { AuthSummary } from 'app/model/user.model';
import { RouterService } from 'app/service/router/router.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { SignoutCurrentUser } from 'app/store/authentication.action';
import { AuthenticationState } from 'app/store/authentication.state';
import { ConfigState } from 'app/store/config.state';
import { HelpState } from 'app/store/help.state';
import { PreferencesState } from 'app/store/preferences.state';
import { Subscription, lastValueFrom } from 'rxjs';
import { filter } from 'rxjs/operators';
import * as actionPreferences from 'app/store/preferences.action';
import { ProjectService } from 'app/service/project/project.service';
import { NzMessageService } from 'ng-zorro-antd/message';
import { ErrorUtils } from 'app/shared/error.utils';
import { V2ProjectService } from 'app/service/projectv2/project.service';
import { Bookmark, BookmarkType } from 'app/model/bookmark.model';
import { BookmarkLoad, BookmarkState } from 'app/store/bookmark.state';

@Component({
    selector: 'app-navbar',
    templateUrl: './navbar.html',
    styleUrls: ['./navbar.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class NavbarComponent implements OnInit, OnDestroy {
    help: Help = new Help();
    loading = true;
    navbarSubscription: Subscription;
    authSubscription: Subscription;
    configSubscription: Subscription;
    recentView = true;
    currentAuthSummary: AuthSummary;
    themeSubscription: Subscription;
    darkActive: boolean;
    projectsSubscription: Subscription;
    workflowsSubscription: Subscription;
    showNotif = false;
    apiConfig: APIConfig;
    selectedProjectKey: string;
    projectSubscription: Subscription;
    projects: Array<Project> = [];
    bookmarks: Array<Bookmark> = [];
    homeActive: boolean;
    bookmarksSubscription: Subscription;

    constructor(
        private _store: Store,
        private _router: Router,
        private _routerService: RouterService,
        private _cd: ChangeDetectorRef,
        private _projectService: ProjectService,
        private _messageService: NzMessageService,
        private _v2ProjectService: V2ProjectService
    ) { }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    changeTheme() {
        this.darkActive = !this.darkActive;
        this._cd.markForCheck();
        this._store.dispatch(new actionPreferences.SetTheme({ theme: this.darkActive ? 'night' : 'light' }));
    }

    ngOnInit() {
        this.authSubscription = this._store.select(AuthenticationState.summary).subscribe(s => {
            this.currentAuthSummary = s;
            this._cd.markForCheck();

            if (s) {
                this.loadProjects();
                this._store.dispatch(new BookmarkLoad());
            }
        });

        this.configSubscription = this._store.select(ConfigState.api).subscribe(c => {
            this.apiConfig = c;
            this._cd.markForCheck();
        });

        this.themeSubscription = this._store.select(PreferencesState.theme).subscribe(t => {
            this.darkActive = t === 'night';
            this._cd.markForCheck();
        });

        this.bookmarksSubscription = this._store.select(BookmarkState.state).subscribe(s => {
            this.bookmarks = s.all;
            this._cd.markForCheck();
        });

        this._store.select(HelpState.last)
            .pipe(
                filter((help) => help != null),
            )
            .subscribe(help => {
                this.help = help;
                this._cd.markForCheck();
            });

        this._router.events.pipe(
            filter(e => e instanceof NavigationEnd),
        ).forEach((e: NavigationEnd) => {
            const params = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root);
            this.selectedProjectKey = params['key'] ?? null;
            this.homeActive = e.url === '/';
            this._cd.markForCheck();
        });

        this.homeActive = this._router.routerState.snapshot.url === '/';
        this._cd.markForCheck();
    }

    async loadProjects() {
        try {
            const res = await Promise.all([
                lastValueFrom(this._projectService.getProjects()),
                lastValueFrom(this._v2ProjectService.getAll())
            ]);
            this.projects = [].concat(res[0])
                .concat(res[1].filter(pv2 => res[0].findIndex(p => p.key === pv2.key) === -1));
            this.projects.sort((a, b) => { return a.name < b.name ? -1 : 1; })
            this._cd.markForCheck();
        } catch (e: any) {
            this._messageService.error(`Unable to load projects: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
    }

    navigateToProject(key): void {
        this._router.navigate(['project/' + key]);
    }

    navigateToWorkflow(key: string, workflowName: string): void {
        this._router.navigate(['project', key, 'workflow', workflowName]);
    }

    clickLogout(): void {
        this._store.dispatch(new SignoutCurrentUser()).subscribe(
            () => {
                this._router.navigate(['/auth/signin']);
            }
        );
    }

    mfaLogin(): void {
        const consumerType = this.currentAuthSummary.consumer.type;
        this._store.dispatch(new SignoutCurrentUser()).subscribe(
            () => {
                this._router.navigate([`/auth/ask-signin/${consumerType}`], {
                    queryParams: {
                        redirect_uri: this._router.url,
                        require_mfa: true
                    }
                });
            }
        );
    }

    generateBookmarkLink(b: Bookmark): Array<string> {
        const splitted = b.id.split('/');
        switch (b.type) {
            case BookmarkType.Workflow:
                const project = splitted.shift();
                return ['/project', project, 'run'];
            case BookmarkType.WorkflowLegacy:
                return ['/project', splitted[0], 'workflow', splitted[1]];
            case BookmarkType.Project:
                return ['/project', b.id];
            default:
                return [];
        }
    }

    generateBookmarkQueryParams(b: Bookmark, variant?: string): any {
        const splitted = b.id.split('/');
        switch (b.type) {
            case BookmarkType.Workflow:
                splitted.shift();
                const workflow_path = splitted.join('/');
                let params = { workflow: workflow_path };
                if (variant) {
                    params['ref'] = variant;
                }
                return params;
            default:
                return {};
        }
    }
}
