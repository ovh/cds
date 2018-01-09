import {Component, Input, OnInit} from '@angular/core';
import {Project} from '../../../../model/project.model';
import {KeyEvent} from '../../../../shared/keys/key.event';
import {ProjectStore} from '../../../../service/project/project.store';
import {finalize} from 'rxjs/operators';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from '@ngx-translate/core';

@Component({
    selector: 'app-project-keys',
    templateUrl: './project.keys.html',
    styleUrls: ['./project.keys.scss']
})
export class ProjectKeysComponent implements OnInit {

    @Input() project: Project;

    loading = false;
    ready = false;

    constructor(private _projectStore: ProjectStore, private _toast: ToastService, private _translate: TranslateService) {
    }

    ngOnInit(): void {
        this._projectStore.getProjectKeysResolver(this.project.key)
            .pipe(finalize(() => this.ready = true))
            .subscribe((proj) => {
                this.project = proj;
            });
    }

    manageKeyEvent(event: KeyEvent): void {
        switch (event.type) {
            case 'add':
                this.loading = true;
                this._projectStore.addKey(this.project.key, event.key).pipe(finalize(() => {
                    this.loading = false;
                })).subscribe(() => this._toast.success('', this._translate.instant('keys_added')));
                break;
            case 'delete':
                this.loading = true;
                this._projectStore.removeKey(this.project.key, event.key.name).pipe(finalize(() => {
                    this.loading = false;
                })).subscribe(() => this._toast.success('', this._translate.instant('keys_removed')))
        }
    }
}
