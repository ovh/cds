import {Component, Input} from '@angular/core';
import {Table} from '../../../../../shared/table/table';
import {Project} from '../../../../../model/project.model';
import {PermissionValue} from '../../../../../model/permission.model';
import {ProjectPlatform} from '../../../../../model/platform.model';
import {ProjectStore} from '../../../../../service/project/project.store';
import {finalize, first} from 'rxjs/operators';
import {TranslateService} from '@ngx-translate/core';
import {ToastService} from '../../../../../shared/toast/ToastService';

@Component({
    selector: 'app-project-platform-list',
    templateUrl: './project.platform.list.html',
    styleUrls: ['./project.platform.list.scss']
})
export class ProjectPlatformListComponent extends Table {

    @Input() project: Project;
    permissionEnum = PermissionValue;
    loading = false;

    constructor(private _projectStore: ProjectStore, private _translate: TranslateService, private _toast: ToastService) {
        super();
    }

    getData(): any[] {
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
