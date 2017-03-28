import {Component, OnInit, ViewChild, Input} from '@angular/core';
import {Application} from '../../../../model/application.model';
import {ApplicationStore} from '../../../../service/application/application.store';
import {TranslateService} from 'ng2-translate';
import {Router} from '@angular/router';
import {ToastService} from '../../../../shared/toast/ToastService';
import {Project} from '../../../../model/project.model';
import {WarningModalComponent} from '../../../../shared/modal/warning/warning.component';

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

    newName: string;
    public loading = false;

    constructor(private _applicationStore: ApplicationStore, private _toast: ToastService,
                public _translate: TranslateService, private _router: Router) {
    }

    ngOnInit() {
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
            this._applicationStore.renameApplication(this.project.key, this.application.name, this.newName).first().subscribe( () => {
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
}
