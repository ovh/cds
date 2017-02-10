import {Component, OnInit, ViewChild, OnDestroy} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {ProjectStore} from '../../../service/project/project.store';
import {Project} from '../../../model/project.model';
import {VariableEvent} from '../../../shared/variable/variable.event.model';
import {ToastService} from '../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {PermissionEvent} from '../../../shared/permission/permission.event.model';
import {Subscription} from 'rxjs/Subscription';
import {WarningModalComponent} from '../../../shared/modal/warning/warning.component';

@Component({
    selector: 'app-project-show',
    templateUrl: './project.html',
    styleUrls: ['./project.scss']
})
export class ProjectShowComponent implements OnInit, OnDestroy {

    public ready = false;
    public varFormLoading = false;
    public permFormLoading = false;

    public project: Project;
    private projectSubscriber: Subscription;

    selectedTab = 'applications';

    @ViewChild('varWarning')
    public varWarningModal: WarningModalComponent;
    @ViewChild('permWarning')
    public permWarningModal: WarningModalComponent;

    constructor(private _projectStore: ProjectStore, private _route: ActivatedRoute, private _router: Router,
                private _toast: ToastService, public _translate: TranslateService) {
    }

    ngOnDestroy(): void {
        if (this.projectSubscriber) {
            this.projectSubscriber.unsubscribe();
        }
    }

    ngOnInit() {
        this._route.params.subscribe(params => {
            const key = params['key'];
            if (key) {
                this.refreshDatas(key);
            }
        });
        this._route.queryParams.subscribe(params => {
            if (params['tab']) {
                this.selectedTab = params['tab'];
            }
        });
    }

    refreshDatas(key: string): void {
        if (this.projectSubscriber) {
            this.projectSubscriber.unsubscribe();
        }
        this.projectSubscriber = this._projectStore.getProjects(key).subscribe( projects => {
            if (projects) {
                const updatedProject = projects.get(key);
                if (updatedProject) {
                    this.project = updatedProject;
                    if (this.project.externalChange) {
                        this._toast.info('', this._translate.instant('warning_project'));
                    }
                    this.ready = true;
                }
            }
        }, () => {
            this._router.navigate(['/home']);
        });
    }

    showTab(tab: string): void {
        this._router.navigateByUrl('/project/' + this.project.key + '?tab=' + tab);
    }

    variableEvent(event: VariableEvent, skip?: boolean): void {
        if (!skip && this.project.externalChange) {
            this.varWarningModal.show(event);
        } else {
            switch (event.type) {
                case 'add':
                    this.varFormLoading = true;
                    this._projectStore.addProjectVariable(this.project.key, event.variable).subscribe(() => {
                        this._toast.success('', this._translate.instant('variable_added'));
                        this.varFormLoading = false;
                    }, () => {
                        this.varFormLoading = false;
                    });
                    break;
                case 'update':
                    this._projectStore.updateProjectVariable(this.project.key, event.variable).subscribe(() => {
                        this._toast.success('', this._translate.instant('variable_updated'));
                    });
                    break;
                case 'delete':
                    this._projectStore.deleteProjectVariable(this.project.key, event.variable).subscribe(() => {
                        this._toast.success('', this._translate.instant('variable_deleted'));
                    });
                    break;
            }
        }
    }

    groupEvent(event: PermissionEvent, skip?: boolean): void {
        if (!skip && this.project.externalChange) {
            this.permWarningModal.show(event);
        } else {
            switch (event.type) {
                case 'add':
                    this.permFormLoading = true;
                    this._projectStore.addProjectPermission(this.project.key, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_added'));
                        this.permFormLoading = false;
                    }, () => {
                        this.permFormLoading = false;
                    });
                    break;
                case 'update':
                    this._projectStore.updateProjectPermission(this.project.key, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_updated'));
                    });
                    break;
                case 'delete':
                    this._projectStore.removeProjectPermission(this.project.key, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_deleted'));
                    });
                    break;
            }
        }
    }
}
