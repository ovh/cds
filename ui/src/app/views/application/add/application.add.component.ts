import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Application } from 'app/model/application.model';
import { Project } from 'app/model/project.model';
import { Variable } from 'app/model/variable.model';
import { VariableService } from 'app/service/variable/variable.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { AddApplication, CloneApplication, FetchApplication } from 'app/store/applications.action';
import { ApplicationsState } from 'app/store/applications.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';
import { finalize, first, flatMap } from 'rxjs/operators';

@Component({
    selector: 'app-application-add',
    templateUrl: './application.add.html',
    styleUrls: ['./application.add.scss']
})
@AutoUnsubscribe()
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
    dataSubscription: Subscription;

    constructor(
        private _activatedRoute: ActivatedRoute,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _router: Router,
        private _varService: VariableService,
        private store: Store
    ) {
        this.dataSubscription = this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });
    }

    ngOnInit(): void {
        this._varService.getContextVariable(this.project.key).pipe(first()).subscribe(s => {
            this.suggestion = s;
        });
    }

    updateSelection(type): void {
        switch (type) {
            case 'clone':
                if (this.project.application_names && this.project.application_names.length > 0) {
                    this.selectedApplicationName = this.project.application_names[0].name;
                    this.updateSelectedApplicationToClone(this.project.application_names[0].name);
                }
                break;
            default:
                this.selectedApplication = undefined;
                break;
        }
    }

    updateSelectedApplicationToClone(appName: string): void {
        this.store.dispatch(new FetchApplication({
            projectKey: this.project.key,
            applicationName: appName
        })).pipe(flatMap(() => this.store.selectOnce(ApplicationsState.selectApplication(this.project.key, appName))))
            .subscribe(app => {
                this.selectedApplication = app;
                this.variables = cloneDeep(app.variables);
                if (this.variables) {
                    this.variables.forEach(v => {
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
                this.store.dispatch(new CloneApplication({
                    projectKey: this.project.key,
                    newApplication,
                    clonedAppName: this.selectedApplication.name
                })).pipe(finalize(() => this.loadingCreate = false))
                    .subscribe(() => {
                        this._toast.success('', this._translate.instant('application_created'));
                        this._router.navigate(['/project', this.project.key, 'application', newApplication.name]);
                    });
                break;

            default:
                this.store.dispatch(new AddApplication({ projectKey: this.project.key, application: newApplication }))
                    .pipe(finalize(() => this.loadingCreate = false))
                    .subscribe(() => {
                        this._toast.success('', this._translate.instant('application_created'));
                        this._router.navigate(['/project', this.project.key, 'application', newApplication.name]);
                    });
        }
    }

    fileEvent(event: { content: string, file: File }) {
        this.fileTooLarge = event.file.size > 100000
        if (this.fileTooLarge) {
            return;
        }
        this.img = event.content;
    }
}
