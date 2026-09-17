import { ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, OnDestroy, OnInit } from '@angular/core';
import { Subscription } from 'rxjs';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { DataEntity, RepositoryAnalysis } from 'app/model/analysis.model';
import { EntityTypeUtil } from 'app/model/entity.model';
import { RepositoryContextService, shortRef } from './repository-context.service';
import { entityOfAnalysisFile } from './entities';

/** One analysis of the repository, with what the page says about it. */
interface AnalysisRow {
    analysis: RepositoryAnalysis;
    registered: number;
    rejected: Array<DataEntity>;
    requestedBy: string;
    /** `event` when a repository event started it, `manual` otherwise. */
    origin: string;
}

const STATUS_BADGE: { [status: string]: string } = {
    Success: 'success',
    Error: 'error',
    Skipped: 'default',
    InProgress: 'processing'
};

/**
 * The analyses of the repository: each read of a ref's `.cds/` folder, what it registered and what
 * it rejected. This is where a definition that does not show up in the entity tabs gets explained.
 */
@Component({
    standalone: false,
    selector: 'app-projectv2-repository-analyses',
    templateUrl: './repository-analyses.html',
    styleUrls: ['./repository-analyses.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2RepositoryAnalysesComponent implements OnInit, OnDestroy {
    ctx = inject(RepositoryContextService);
    /** All the analyses, most recent first; null until read. */
    rows: Array<AnalysisRow> = null;
    filtered: Array<AnalysisRow> = [];
    refs: Array<string> = [];
    refFilter: string = null;
    statusFilter: string = null;
    expanded = new Set<string>();
    loading: boolean = false;

    analysesSub: Subscription;

    private _cd = inject(ChangeDetectorRef);

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.analysesSub = this.ctx.analyses$.subscribe(analyses => {
            this.buildRows(analyses);
            this._cd.markForCheck();
        });
    }

    private buildRows(analyses: Array<RepositoryAnalysis>): void {
        if (analyses === null) {
            this.rows = null;
            this.filtered = [];
            return;
        }
        this.rows = analyses
            .map(analysis => ({
                analysis,
                registered: (analysis.data?.entities ?? []).filter(e => e.status === 'Success').length,
                rejected: (analysis.data?.entities ?? []).filter(e => e.status !== 'Success'),
                requestedBy: this.requestedBy(analysis),
                origin: analysis.data?.hook_event_uuid ? 'event' : 'manual'
            }))
            .sort((a, b) => Date.parse(b.analysis.created) - Date.parse(a.analysis.created));
        this.refs = [...new Set(this.rows.map(r => r.analysis.ref))].sort();
        this.applyFilters();
    }

    /** Who asked: the CDS user of the initiator, else its vcs user, else what older analyses carry. */
    private requestedBy(analysis: RepositoryAnalysis): string {
        const initiator = analysis.data?.initiator;
        return initiator?.user?.username || initiator?.vcs_username || analysis.data?.cds_username || null;
    }

    applyFilters(): void {
        this.filtered = (this.rows ?? [])
            .filter(r => !this.refFilter || r.analysis.ref === this.refFilter)
            .filter(r => !this.statusFilter || r.analysis.status === this.statusFilter);
    }

    changeRefFilter(ref: string): void {
        this.refFilter = ref || null;
        this.applyFilters();
        this._cd.markForCheck();
    }

    changeStatusFilter(status: string): void {
        this.statusFilter = status || null;
        this.applyFilters();
        this._cd.markForCheck();
    }

    async refresh() {
        this.loading = true;
        this._cd.markForCheck();
        await this.ctx.reloadAnalyses();
        this.loading = false;
        this._cd.markForCheck();
    }

    toggle(row: AnalysisRow): void {
        if (this.expanded.has(row.analysis.id)) {
            this.expanded.delete(row.analysis.id);
        } else {
            this.expanded.add(row.analysis.id);
        }
        this._cd.markForCheck();
    }

    isExpanded(row: AnalysisRow): boolean {
        return this.expanded.has(row.analysis.id);
    }

    /** Where a file of the analysis is read in this page, on the ref of the analysis. */
    fileLink(file: DataEntity): Array<string> {
        const entity = entityOfAnalysisFile(file.path, file.file_name);
        if (!entity) {
            return null;
        }
        return ['/project', this.ctx.project.key, 'explore', 'vcs', this.ctx.vcsName, 'repository', this.ctx.repoName, EntityTypeUtil.toURLParam(entity.type), entity.name];
    }

    badge(status: string): string {
        return STATUS_BADGE[status] ?? 'default';
    }

    isTag(ref: string): boolean {
        return (ref ?? '').startsWith('refs/tags/');
    }

    shortRef(ref: string): string {
        return shortRef(ref);
    }
}
