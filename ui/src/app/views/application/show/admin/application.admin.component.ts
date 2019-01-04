import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {Router} from '@angular/router';
import {TranslateService} from '@ngx-translate/core';
import {cloneDeep} from 'lodash';
import {first} from 'rxjs/operators';
import {Application} from '../../../../model/application.model';
import {Project} from '../../../../model/project.model';
import {User} from '../../../../model/user.model';
import {ApplicationStore} from '../../../../service/application/application.store';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {WarningModalComponent} from '../../../../shared/modal/warning/warning.component';
import {ToastService} from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-application-admin',
    templateUrl: './application.admin.html',
    styleUrls: ['./application.admin.scss']
})
export class ApplicationAdminComponent implements OnInit {

    @Input() application: Application;
    @Input() project: Project;
    @ViewChild('updateWarning')
        private updateWarningModal: WarningModalComponent;

    user: User;

    newName: string;
    fileTooLarge = false;
    public loading = false;

    constructor(private _applicationStore: ApplicationStore, private _toast: ToastService,
                public _translate: TranslateService, private _router: Router, private _authStore: AuthentificationStore) {
    }

    ngOnInit() {
        this.user = this._authStore.getUser();
        this.newName = this.application.name;
        if (this.application.permission !== 7) {
            this._router.navigate(['/project', this.project.key, 'application', this.application.name],
                { queryParams: {tab: 'workflow'}});
        }
    }

    onSubmitApplicationUpdate(skip?: boolean): void {
        if (!skip && this.application.externalChange) {
            this.updateWarningModal.show();
        } else {
            this.loading = true;
            let app = cloneDeep(this.application);
            app.name = this.newName;
            this._applicationStore.updateApplication(this.project.key, this.application.name, app)
                .pipe(first()).subscribe( () => {
                this.loading = false;
                this._toast.success('', this._translate.instant('application_update_ok'));
                this._router.navigate(['/project', this.project.key, 'application', this.newName]);
            }, () => {
                this.loading = false;
            });
        }
    }

    deleteApplication(): void {
        this.loading = true;
        this._applicationStore.deleteApplication(this.project.key, this.application.name).subscribe(() => {
            this.loading = false;
            this._toast.success('', this._translate.instant('application_deleted'));
            this._router.navigate(['/project', this.project.key]);
        }, () => {
            this.loading = false;
        });
    }

    fileEvent(event: {content: string, file: File}) {
        this.fileTooLarge = event.file.size > 100000;
        if (this.fileTooLarge) {
            return;
        }
        this.application.icon = event.content;
    }
}
