import { ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { NzDrawerService } from 'ng-zorro-antd/drawer';
import { NzMessageService } from 'ng-zorro-antd/message';
import { combineLatest, Subscription } from 'rxjs';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ErrorUtils } from 'app/shared/error.utils';
import { ProjectRepository } from 'app/model/project.model';
import { RepositoryContextService } from './repository-context.service';
import { ProjectV2RunStartComponent, ProjectV2RunStartComponentParams } from '../../run-start/run-start.component';
import { ProjectV2TriggerAnalysisComponent, ProjectV2TriggerAnalysisComponentParams } from '../trigger-analysis/trigger-analysis.component';

/** A tab of the repository page: a child route and its label. */
export interface RepositoryTab {
    path: string;
    label: string;
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
    childActive: boolean = false;
    error: string;

    routeSub: Subscription;
    repositorySub: Subscription;

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
        this.repositorySub = this.ctx.repository$.subscribe(repository => {
            this.tabs = this.tabsFor(repository);
            this._cd.markForCheck();
        });
    }

    async load(vcsName: string, repoName: string, ref: string) {
        this.error = null;
        try {
            await this.ctx.load(vcsName, repoName, ref);
        } catch (e: any) {
            this.error = ErrorUtils.print(e);
            this._messageService.error(`Unable to load repository: ${this.error}`, { nzDuration: 2000 });
        }
        this._cd.markForCheck();
    }

    /** The tabs a repository offers; each one exists once its child route does. */
    private tabsFor(repository: ProjectRepository): Array<RepositoryTab> {
        if (!repository) {
            return [];
        }
        return [];
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
        return (ref ?? '').replace(/^refs\/(heads|tags)\//, '');
    }
}
