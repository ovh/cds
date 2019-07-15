import { Component, Input, OnInit, ViewChild } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AuthenticationState } from 'app/store/authentication.state';
import { DeleteProject, UpdateProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';
import { Project } from '../../../../model/project.model';
import { User } from '../../../../model/user.model';
import { Warning } from '../../../../model/warning.model';
import { WarningModalComponent } from '../../../../shared/modal/warning/warning.component';
import { ToastService } from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-project-admin',
    templateUrl: './project.admin.html',
    styleUrls: ['./project.admin.scss']
})
export class ProjectAdminComponent implements OnInit {

    @Input('warnings')
    set warnings(data: Array<Warning>) {
        if (data) {
            this.unusedWarning = new Map<string, Warning>();
            this.missingWarnings = new Array<Warning>();
            data.forEach(v => {
                if (v.type.indexOf('MISSING') !== -1) {
                    this.missingWarnings.push(v);
                } else {
                    this.unusedWarning.set(v.element, v);
                }
            });
        }
    };
    missingWarnings: Array<Warning>;
    unusedWarning: Map<string, Warning>;

    @Input() project: Project;
    @ViewChild('updateWarning', {static: false})
    private warningUpdateModal: WarningModalComponent;

    loading = false;
    fileTooLarge = false;
    user: User;

    constructor(
        private _toast: ToastService,
        public _translate: TranslateService,
        private _router: Router,
        private _store: Store
    ) { };

    ngOnInit(): void {
        if (!this.project.permissions.writable) {
            this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'applications' } });
        }
        this.user = this._store.selectSnapshot(AuthenticationState.user);
    }

    onSubmitProjectUpdate(skip?: boolean) {
        if (!skip && this.project.externalChange) {
            this.warningUpdateModal.show();
        } else {
            this.loading = true;
            this._store.dispatch(new UpdateProject(this.project))
                .pipe(finalize(() => this.loading = false))
                .subscribe(() => this._toast.success('', this._translate.instant('project_update_msg_ok')));
        }
    };

    deleteProject(): void {
        this.loading = true;
        this._store.dispatch(new DeleteProject({ projectKey: this.project.key }))
            .pipe(finalize(() => this.loading = false))
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
