import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { ProjectIntegration } from 'app/model/integration.model';
import { Project } from 'app/model/project.model';
import { ThemeStore } from 'app/service/theme/theme.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Table } from 'app/shared/table/table';
import { ToastService } from 'app/shared/toast/ToastService';
import { DeleteIntegrationInProject, UpdateIntegrationInProject } from 'app/store/project.action';
import { Subscription } from 'rxjs';
import { finalize, first } from 'rxjs/operators';

@Component({
    selector: 'app-project-integration-list',
    templateUrl: './project.integration.list.html',
    styleUrls: ['./project.integration.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectIntegrationListComponent extends Table<ProjectIntegration> implements OnInit, OnDestroy {
    @ViewChild('codeMirror') codemirror: any;

    @Input() project: Project;

    loading = false;
    codeMirrorConfig: any;
    themeSubscription: Subscription;

    constructor(
        private _translate: TranslateService,
        private _toast: ToastService,
        private store: Store,
        private _theme: ThemeStore,
        private _cd: ChangeDetectorRef
    ) {
        super();
        this.codeMirrorConfig = {
            mode: 'shell',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true
        };
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.themeSubscription = this._theme.get().subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
            this._cd.markForCheck();
        });
    }

    getData(): Array<ProjectIntegration> {
        return this.project.integrations;
    }

    deleteIntegration(p: ProjectIntegration): void {
        this.loading = true;
        this.store.dispatch(new DeleteIntegrationInProject({
            projectKey: this.project.key,
            integration: p
        })).pipe(first(), finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        }))
            .subscribe(() => this._toast.success('', this._translate.instant('project_updated')));
    }

    updateIntegration(p: ProjectIntegration): void {
        this.loading = true;
        this.store.dispatch(new UpdateIntegrationInProject({
            projectKey: this.project.key,
            integrationName: p.name,
            changes: p
        })).pipe(first(), finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        }))
            .subscribe(() => this._toast.success('', this._translate.instant('project_updated')));
    }
}
