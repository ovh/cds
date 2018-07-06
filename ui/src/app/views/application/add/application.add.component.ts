import {Component, OnInit} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {TranslateService} from '@ngx-translate/core';
import {cloneDeep} from 'lodash';
import {first} from 'rxjs/operators';
import {Application} from '../../../model/application.model';
import {Project} from '../../../model/project.model';
import {Variable} from '../../../model/variable.model';
import {ApplicationStore} from '../../../service/application/application.store';
import {ProjectStore} from '../../../service/project/project.store';
import {VariableService} from '../../../service/variable/variable.service';
import {ToastService} from '../../../shared/toast/ToastService';

@Component({
    selector: 'app-application-add',
    templateUrl: './application.add.html',
    styleUrls: ['./application.add.scss']
})
export class ApplicationAddComponent implements OnInit {

    ready = false;
    project: Project;
    typeofCreation = 'empty';

    selectedName: string;
    description: string;
    variables: Array<Variable>;
    selectedApplication: Application;
    selectedApplicationName: string;

    loadingCreate = false;

    applicationNamePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');
    appPatternError = false;

    suggestion: Array<string>;
    img: string;
    fileTooLarge = false;

    constructor(private _activatedRoute: ActivatedRoute, private _projectStore: ProjectStore,
                private _appStore: ApplicationStore, private _toast: ToastService, private _translate: TranslateService,
                private _router: Router, private _varService: VariableService) {
        this._activatedRoute.data.subscribe( datas => {
            this.project = datas['project'];
        });
    }

    ngOnInit(): void {
        this._varService.getContextVariable(this.project.key).pipe(first()).subscribe( s => {
            this.suggestion = s;
        });
        if (!this.project.applications) {
            this._projectStore.getProjectApplicationsResolver(this.project.key).pipe(first()).subscribe(proj => {
                this.project = proj;
            });
        }
    }

    updateSelection(type): void {
        switch (type) {
            case 'clone':
                if (this.project.applications && this.project.applications.length > 0) {
                    this.selectedApplicationName = this.project.applications[0].name;
                    this.updateSelectedApplicationToClone(this.project.applications[0].name);
                }
                break;
            default:
                this.selectedApplication = undefined;
                break;
        }
    }

    updateSelectedApplicationToClone(appName: string): void {
        this._appStore.getApplicationResolver(this.project.key, appName).pipe(first()).subscribe(app => {
            this.selectedApplication = app;
            this.variables = cloneDeep(app.variables);
            if (this.variables) {
                this.variables.forEach( v => {
                    if (v.type === 'password') {
                        v.value = '';
                    }
                });
            }
        });
    }

    createApplication(): void {
        if (this.selectedName) {
            if (!this.applicationNamePattern.test(this.selectedName)) {
                this.appPatternError = true;
                return;
            }
        }

        if (this.variables) {
            this.variables.forEach(v => {
                v.value = String(v.value);
            });
        }

        let newApplication = new Application();
        newApplication.name = this.selectedName;
        newApplication.description = this.description;
        newApplication.icon = this.img;

        this.loadingCreate = true;
        switch (this.typeofCreation) {
            case 'clone':
                newApplication.variables = this.variables;
                this._appStore.cloneApplication(this.project.key, this.selectedApplication.name, newApplication).subscribe(() => {
                    this.loadingCreate = false;
                    this._toast.success('', this._translate.instant('application_created'));
                    this._router.navigate(['/project', this.project.key, 'application', newApplication.name]);
                }, () => {
                    this.loadingCreate = false;
                });
                break;

            default:
                this._appStore.createApplication(this.project.key, newApplication).subscribe(() => {
                    this.loadingCreate = false;
                    this._toast.success('', this._translate.instant('application_created'));
                    this._router.navigate(['/project', this.project.key, 'application', newApplication.name]);
                }, () => {
                    this.loadingCreate = false;
                });
        }
    }

    fileEvent(event: {content: string, file: File}) {
        this.fileTooLarge = event.file.size > 100000
        if (this.fileTooLarge) {
          return;
        }
        this.img = event.content;
    }
}
