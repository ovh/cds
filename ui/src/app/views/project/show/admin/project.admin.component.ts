import {Input, ViewChild, OnInit, Component} from '@angular/core';
import {Project} from '../../../../model/project.model';
import {ProjectStore} from '../../../../service/project/project.store';
import {TranslateService} from 'ng2-translate';
import {ToastService} from '../../../../shared/toast/ToastService';
import {WarningModalComponent} from '../../../../shared/modal/warning/warning.component';
import {Router} from '@angular/router';

@Component({
    selector: 'app-project-admin',
    templateUrl: './project.admin.html',
    styleUrls: ['./project.admin.scss']
})
export class ProjectAdminComponent implements OnInit {

    @Input() project: Project;
    @ViewChild('updateWarning')
        private warningUpdateModal: WarningModalComponent;

    loading = false;

    constructor(private _projectStore: ProjectStore, private _toast: ToastService,
                public _translate: TranslateService, private _router: Router) {};

    ngOnInit(): void {
        if (this.project.permission !== 7) {
            this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'applications' }});
        }
    }

    onSubmitProjectUpdate(skip?: boolean) {
        if (!skip && this.project.externalChange) {
            this.warningUpdateModal.show();
        } else {
            this.loading = true;
            this._projectStore.updateProject(this.project).subscribe(() => {
                this.loading = false;
                this._toast.success('', this._translate.instant('project_update_msg_ok') );
            }, () => {
                this.loading = false;
            });
        }
    };

    deleteProject(): void {
        this._projectStore.deleteProject(this.project.key).subscribe(() => {
            this.loading = false;
            this._toast.success('', this._translate.instant('project_deleted'));
            this._router.navigate(['/home']);
        }, () => {
            this.loading = false;
        });
    }

}
