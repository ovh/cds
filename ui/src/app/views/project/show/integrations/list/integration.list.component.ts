import { Component, Input, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { DeleteIntegrationInProject, UpdateIntegrationInProject } from 'app/store/project.action';
import { CodemirrorComponent } from 'ng2-codemirror-typescript/Codemirror';
import { finalize, first } from 'rxjs/operators';
import { ProjectIntegration } from '../../../../../model/integration.model';
import { PermissionValue } from '../../../../../model/permission.model';
import { Project } from '../../../../../model/project.model';
import { Table } from '../../../../../shared/table/table';
import { ToastService } from '../../../../../shared/toast/ToastService';

@Component({
    selector: 'app-project-integration-list',
    templateUrl: './project.integration.list.html',
    styleUrls: ['./project.integration.list.scss']
})
export class ProjectIntegrationListComponent extends Table<ProjectIntegration> {

    @Input() project: Project;
    @ViewChild('codeMirror')
    codemirror: CodemirrorComponent;
    permissionEnum = PermissionValue;
    loading = false;
    codeMirrorConfig: {};

    constructor(
        private _translate: TranslateService,
        private _toast: ToastService,
        private store: Store
    ) {
        super();
        this.codeMirrorConfig = {
            mode: 'shell',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true
        };
    }

    getData(): Array<ProjectIntegration> {
        return this.project.integrations;
    }

    deleteIntegration(p: ProjectIntegration): void {
        this.loading = true;
        this.store.dispatch(new DeleteIntegrationInProject({
            projectKey: this.project.key,
            integration: p
        })).pipe(first(), finalize(() => this.loading = false))
            .subscribe(() => this._toast.success('', this._translate.instant('project_updated')));
    }

    updateIntegration(p: ProjectIntegration): void {
        this.loading = true;
        this.store.dispatch(new UpdateIntegrationInProject({
            projectKey: this.project.key,
            integrationName: p.name,
            changes: p
        })).pipe(first(), finalize(() => this.loading = false))
            .subscribe(() => this._toast.success('', this._translate.instant('project_updated')));
    }
}
