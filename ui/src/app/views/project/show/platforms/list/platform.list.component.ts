import { Component, Input, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { CodemirrorComponent } from 'ng2-codemirror-typescript/Codemirror';
import { finalize, first } from 'rxjs/operators';
import { PermissionValue } from '../../../../../model/permission.model';
import { ProjectPlatform } from '../../../../../model/platform.model';
import { Project } from '../../../../../model/project.model';
import { ProjectStore } from '../../../../../service/project/project.store';
import { Table } from '../../../../../shared/table/table';
import { ToastService } from '../../../../../shared/toast/ToastService';

@Component({
    selector: 'app-project-platform-list',
    templateUrl: './project.platform.list.html',
    styleUrls: ['./project.platform.list.scss']
})
export class ProjectPlatformListComponent extends Table<ProjectPlatform> {

    @Input() project: Project;
    @ViewChild('codeMirror')
    codemirror: CodemirrorComponent;
    permissionEnum = PermissionValue;
    loading = false;
    codeMirrorConfig: {};

    constructor(private _projectStore: ProjectStore, private _translate: TranslateService, private _toast: ToastService) {
        super();
        this.codeMirrorConfig = {
            mode: 'shell',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true
        };
    }

    getData(): Array<ProjectPlatform> {
        return this.project.platforms;
    }

    deletePlatform(p: ProjectPlatform): void {
        this.loading = true;
        this._projectStore.deleteProjectPlatform(this.project.key, p.name).pipe(first(), finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('project_updated'));
            });
    }

    updatePlatform(p: ProjectPlatform): void {
        this.loading = true;
        this._projectStore.updateProjectPlatform(this.project.key, p).pipe(first(), finalize(() => this.loading = false)).subscribe(() => {
            this._toast.success('', this._translate.instant('project_updated'));
        });
    }
}
