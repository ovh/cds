import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, ViewChild } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Project } from 'app/model/project.model';
import { WarningModalComponent } from 'app/shared/modal/warning/warning.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { DeleteProject, UpdateProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-project-admin',
    templateUrl: './project.admin.html',
    styleUrls: ['./project.admin.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectAdminComponent implements OnInit {

    @Input() project: Project;
    @ViewChild('updateWarning')
    private warningUpdateModal: WarningModalComponent;

    loading = false;
    fileTooLarge = false;

    constructor(
        private _toast: ToastService,
        public _translate: TranslateService,
        private _router: Router,
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit(): void {
        if (!this.project.permissions.writable) {
            this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'applications' } });
        }
    }

    onSubmitProjectUpdate(skip?: boolean) {
        if (!skip && this.project.externalChange) {
            this.warningUpdateModal.show();
        } else {
            this.loading = true;
            this._store.dispatch(new UpdateProject(this.project))
                .pipe(finalize(() => {
                    this.loading = false;
                    this._cd.markForCheck();
                }))
                .subscribe(() => this._toast.success('', this._translate.instant('project_update_msg_ok')));
        }
    }

    deleteProject(): void {
        this.loading = true;
        this._store.dispatch(new DeleteProject({ projectKey: this.project.key }))
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('project_deleted'));
                this._router.navigate(['/home']);
            });
    }

    fileEvent(event: { content: string, file: File }) {
        this.fileTooLarge = event.file.size > 100000;
        if (this.fileTooLarge) {
            return;
        }
        this.project.icon = event.content;
    }
}
