import { ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { NzDrawerService } from 'ng-zorro-antd/drawer';
import { Subscription } from 'rxjs';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Entity, EntityType, EntityTypeUtil } from 'app/model/entity.model';
import { RepositoryContextService, shortRef } from './repository-context.service';
import { ENTITY_TYPE_LABELS } from './entities';
import { ProjectV2TriggerAnalysisComponent, ProjectV2TriggerAnalysisComponentParams } from '../trigger-analysis/trigger-analysis.component';

/**
 * The tab of one entity type: the entities of the current ref on the left, the selected one on the
 * right. The selection lives in the url, so that a link to an entity opens the tab on it.
 */
@Component({
    standalone: false,
    selector: 'app-projectv2-repository-entities',
    templateUrl: './repository-entities.html',
    styleUrls: ['./repository-entities.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2RepositoryEntitiesComponent implements OnInit, OnDestroy {
    ctx = inject(RepositoryContextService);
    type: EntityType;
    typeLabel: string;
    typeParam: string;
    /** The entities of the type on the current ref; null until read. */
    entities: Array<Entity> = null;
    filtered: Array<Entity> = [];
    filter: string = '';
    selectedName: string;
    selected: Entity;

    routeSub: Subscription;
    entitiesSub: Subscription;

    private _route = inject(ActivatedRoute);
    private _router = inject(Router);
    private _cd = inject(ChangeDetectorRef);
    private _drawerService = inject(NzDrawerService);

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.routeSub = this._route.paramMap.subscribe(params => {
            this.type = EntityTypeUtil.fromURLParam(params.get('entityType'));
            this.typeParam = EntityTypeUtil.toURLParam(this.type);
            this.typeLabel = ENTITY_TYPE_LABELS[this.type];
            this.selectedName = params.get('entityName');
            this.apply();
        });
        this.entitiesSub = this.ctx.entities$.subscribe(() => this.apply());
    }

    private apply(): void {
        const groups = this.ctx.entities$.value;
        this.entities = groups ? groups.get(this.type) ?? [] : null;
        this.applyFilter();
        this.selected = this.entities?.find(e => e.name === this.selectedName) ?? null;
        // Without a valid entity in the url, the first one opens. Replacing the url means every
        // later selection only changes params, and this component stays mounted.
        if (this.entities?.length && !this.selected) {
            this.open(this.entities[0], true);
        }
        this._cd.markForCheck();
    }

    applyFilter(): void {
        const needle = this.filter.trim().toLowerCase();
        this.filtered = (this.entities ?? []).filter(e => !needle
            || e.name.toLowerCase().includes(needle)
            || (e.file_path ?? '').toLowerCase().includes(needle));
    }

    changeFilter(value: string): void {
        this.filter = value;
        this.applyFilter();
        this._cd.markForCheck();
    }

    open(entity: Entity, replaceUrl: boolean = false): void {
        this._router.navigate(this.linkTo(entity), { replaceUrl, queryParamsHandling: 'preserve' });
    }

    linkTo(entity: Entity): Array<string> {
        return ['/project', this.ctx.project.key, 'explore', 'vcs', this.ctx.vcsName, 'repository', this.ctx.repoName, this.typeParam, entity.name];
    }

    shortRef(ref: string): string {
        return shortRef(ref);
    }

    /** The ref lives in the url; the page reacts to it like to any navigation. */
    switchToDefaultRef(): void {
        const ref = this.ctx.defaultRef();
        this.ctx.rememberRef(ref);
        this._router.navigate([], { queryParams: { ref }, queryParamsHandling: 'merge' });
    }

    analyzeRef(): void {
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
}
