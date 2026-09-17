import { ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, NavigationEnd, Router } from '@angular/router';
import { NzDrawerService } from 'ng-zorro-antd/drawer';
import { NzMessageService } from 'ng-zorro-antd/message';
import { combineLatest, filter, Subscription } from 'rxjs';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ProjectRepository } from 'app/model/project.model';
import { EntityType, EntityTypeUtil } from 'app/model/entity.model';
import { apiErrorMessage, RepositoryContextService, shortRef } from './repository-context.service';
import { ENTITY_TYPE_LABELS, ENTITY_TYPE_ORDER } from './entities';
import { ProjectV2RunStartComponent, ProjectV2RunStartComponentParams } from '../../run-start/run-start.component';
import { ProjectV2TriggerAnalysisComponent, ProjectV2TriggerAnalysisComponentParams } from '../trigger-analysis/trigger-analysis.component';

/** A tab of the repository page: a child route, its label and, for an entity type, how many it holds. */
export interface RepositoryTab {
    path: string;
    label: string;
    count?: number;
}

/**
 * The page of a repository: a header (name, ref, actions) and tabs, each tab being a child route.
 * A repository the project only listens to has no ref nor actions, only its events.
 */
@Component({
    standalone: false,
    selector: 'app-projectv2-repository',
    templateUrl: './repository.html',
    styleUrls: ['./repository.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush,
    providers: [RepositoryContextService]
})
@AutoUnsubscribe()
export class ProjectV2RepositoryComponent implements OnInit, OnDestroy {
    ctx = inject(RepositoryContextService);
    tabs: Array<RepositoryTab> = [];
    /** True once everything the header shows is known, so that it appears in one go. */
    ready: boolean = false;
    childActive: boolean = false;
    error: any;

    routeSub: Subscription;
    readySub: Subscription;
    tabsSub: Subscription;
    navigationSub: Subscription;

    private _route = inject(ActivatedRoute);
    private _router = inject(Router);
    private _cd = inject(ChangeDetectorRef);
    private _drawerService = inject(NzDrawerService);
    private _messageService = inject(NzMessageService);

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.routeSub = combineLatest([this._route.paramMap, this._route.queryParamMap]).subscribe(([params, query]) => {
            this.load(params.get('vcsName'), params.get('repoName'), query.get('ref'));
        });

        // Subscribe in case of repository change (from sidebar)
        this.readySub = combineLatest([this.ctx.repository$, this.ctx.ref$, this.ctx.listenedBy$]).subscribe(([repository, ref, listenedBy]) => {
            this.ready = !!repository && (repository.distant ? listenedBy !== null : ref !== null);
            this._cd.markForCheck();
        });
        this.tabsSub = combineLatest([this.ctx.repository$, this.ctx.entities$]).subscribe(([repository]) => {
            this.tabs = this.tabsFor(repository);
            this.openDefaultTab();
            this._cd.markForCheck();
        });
        // Reaching the bare repository url while the page is already shown changes no param
        this.navigationSub = this._router.events.pipe(filter(e => e instanceof NavigationEnd)).subscribe(() => this.openDefaultTab());
    }

    async load(vcsName: string, repoName: string, ref: string) {
        this.error = null;
        try {
            await this.ctx.load(vcsName, repoName, ref);
        } catch (e: any) {
            this.error = e;
            this._messageService.error(`Unable to load repository: ${this.errorMessage(e)}`, { nzDuration: 2000 });
        }
        this._cd.markForCheck();
    }

    errorMessage(e: any): string {
        return apiErrorMessage(e);
    }

    /** The tabs a repository offers; a listened one has no entities, only its activity. */
    private tabsFor(repository: ProjectRepository): Array<RepositoryTab> {
        if (!repository) {
            return [];
        }
        const activity: RepositoryTab = { path: 'activity', label: 'Activity' };
        if (repository.distant) {
            return [activity];
        }
        const entityTabs: Array<RepositoryTab> = ENTITY_TYPE_ORDER
            .map(type => ({ path: EntityTypeUtil.toURLParam(type), label: ENTITY_TYPE_LABELS[type], count: this.ctx.entityCount(type) }))
            // Jobs are seldom defined: their tab only shows up when the ref has some
            .filter(tab => tab.count > 0 || tab.path !== EntityTypeUtil.toURLParam(EntityType.Job));
        return [...entityTabs, activity];
    }

    /** Without a tab in the url, the first one opens, the url being replaced rather than stacked. */
    private openDefaultTab(): void {
        if (this._route.firstChild || this.tabs.length === 0) {
            return;
        }
        this._router.navigate([this.tabs[0].path], { relativeTo: this._route, replaceUrl: true, queryParamsHandling: 'preserve' });
    }

    /** The ref lives in the url; the page reacts to it like to any navigation. */
    changeRef(ref: string): void {
        this.ctx.rememberRef(ref);
        this._router.navigate([], { queryParams: { ref }, queryParamsHandling: 'merge' });
    }

    openRunStartDrawer(): void {
        this._drawerService.create<ProjectV2RunStartComponent, { params: ProjectV2RunStartComponentParams }, string>({
            nzTitle: 'Start new Workflow Run',
            nzContent: ProjectV2RunStartComponent,
            nzContentParams: {
                params: <ProjectV2RunStartComponentParams>{
                    workflow_repository: this.ctx.repositoryPath,
                    workflow_ref: this.ctx.ref
                }
            },
            nzSize: 'large',
            nzBodyStyle: { 'padding': '0' }
        });
    }

    openTriggerAnalysisDrawer(): void {
        this._drawerService.create<ProjectV2TriggerAnalysisComponent, { params: ProjectV2TriggerAnalysisComponentParams }, string>({
            nzTitle: 'Trigger repository analysis',
            nzContent: ProjectV2TriggerAnalysisComponent,
            nzContentParams: {
                params: <ProjectV2TriggerAnalysisComponentParams>{
                    repository: this.ctx.repositoryPath,
                    ref: this.ctx.ref
                }
            },
            nzSize: 'large'
        });
    }

    shortRef(ref: string): string {
        return shortRef(ref);
    }
}
