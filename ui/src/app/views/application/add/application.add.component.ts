import {Component, OnInit} from '@angular/core';
import {ApplyTemplateRequest, Template} from '../../../model/template.model';
import {ApplicationTemplateService} from '../../../service/application/application.template.service';
import {ActivatedRoute, Router} from '@angular/router';
import {Project} from '../../../model/project.model';
import {Application} from '../../../model/application.model';
import {Variable} from '../../../model/variable.model';
import {ApplicationStore} from '../../../service/application/application.store';
import {Parameter} from '../../../model/parameter.model';
import {TranslateService} from '@ngx-translate/core';
import {ToastService} from '../../../shared/toast/ToastService';
import {VariableService} from '../../../service/variable/variable.service';
import {cloneDeep} from 'lodash';
import {first} from 'rxjs/operators';

@Component({
    selector: 'app-application-add',
    templateUrl: './application.add.html',
    styleUrls: ['./application.add.scss']
})
export class ApplicationAddComponent implements OnInit {

    ready = false;
    project: Project;
    templates: Array<Template>;
    typeofCreation = 'empty';

    selectedName: string;
    parameters: Array<Parameter>;
    variables: Array<Variable>;
    selectedTemplate: Template;
    selectedApplication: Application;
    selectedApplicationName: string;

    loadingCreate = false;

    applicationNamePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');
    appPatternError = false;

    suggestion: Array<string>;

    constructor(private _appTemplateService: ApplicationTemplateService, private _activatedRoute: ActivatedRoute,
                private _appStore: ApplicationStore, private _toast: ToastService, private _translate: TranslateService,
                private _router: Router, private _varService: VariableService) {
        this._activatedRoute.data.subscribe( datas => {
            this.project = datas['project'];
        });

        this._appTemplateService.getTemplates().subscribe(templates => {
            this.ready = true;
            this.templates = templates;
        });
    }

    ngOnInit(): void {
        this._varService.getContextVariable(this.project.key).pipe(first()).subscribe( s => {
            this.suggestion = s;
        });
    }

    updateSelection(type): void {
        switch (type) {
            case 'clone':
                this.selectedTemplate = undefined;
                if (this.project.applications && this.project.applications.length > 0) {
                    this.selectedApplicationName = this.project.applications[0].name;
                    this.updateSelectedApplicationToClone(this.project.applications[0].name);
                }
                break;
            default:
                this.selectedApplication = undefined;
                if (type === 'empty') {
                    this.updateSelectedTemplateToUse('Void');
                } else {
                    if (this.templates && this.templates.length > 0) {
                        this.updateSelectedTemplateToUse(this.templates[0].name);
                    }
                }
                break;
        }
    }

    updateSelectedTemplateToUse(name: string): void {
        this.variables = null;
        this.selectedTemplate = this.templates.find( t => t.name === name);
        if (this.selectedTemplate) {
            this.parameters = this.selectedTemplate.params;
        } else {
            this.selectedTemplate = new Template();
            this.selectedTemplate.name = name;
            this.parameters = null;
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
        if (this.parameters) {
            this.parameters.forEach(p => {
                p.value = String(p.value);
            });
        }

        this.loadingCreate = true;
        switch (this.typeofCreation) {
            case 'clone':
                let newApplication = new Application();
                newApplication.name = this.selectedName;
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
                let addAppRequest: ApplyTemplateRequest = new ApplyTemplateRequest();
                addAppRequest.name = this.selectedName;

                if (this.typeofCreation === 'empty') {
                    addAppRequest.template = 'Void';
                } else {
                    addAppRequest.template = this.selectedTemplate.name;
                    addAppRequest.template_params = this.parameters;
                }

                this._appStore.applyTemplate(this.project.key, addAppRequest).subscribe(() => {
                    this.loadingCreate = false;
                    this._toast.success('', this._translate.instant('application_created'));
                    this._router.navigate(['/project', this.project.key, 'application', addAppRequest.name]);
                }, () => {
                    this.loadingCreate = false;
                });
        }
    }
}
