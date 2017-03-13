import {NgModule, ModuleWithProviders, SkipSelf, Optional} from '@angular/core';
import {ProjectService} from './project/project.service';
import {ProjectStore} from './project/project.store';
import {RequestOptions, XHRBackend, Http} from '@angular/http';
import {HttpService} from './http-service.service';
import {AuthentificationStore} from './auth/authentification.store';
import {UserService} from './user/user.service';
import {CanActivateAuthRoute} from './auth/authenRouteActivate';
import {Router} from '@angular/router';
import {WarningStore} from './warning/warning.store';
import {PipelineStore} from './pipeline/pipeline.store';
import {PipelineService} from './pipeline/pipeline.service';
import {ApplicationService} from './application/application.service';
import {ApplicationStore} from './application/application.store';
import {ApplicationPipelineService} from './application/pipeline/application.pipeline.service';
import {VariableService} from './variable/variable.service';
import {GroupService} from './group/group.service';
import {RepoManagerService} from './repomanager/project.repomanager.service';
import {ToastService} from '../shared/toast/ToastService';
import {ApplicationWorkflowService} from './application/application.workflow.service';
import {RequirementService} from './worker/requirement/requirement.service';
import {RequirementStore} from './worker/requirement/requirement.store';
import {ParameterService} from './parameter/parameter.service';
import {ActionService} from './action/action.service';
import {ActionStore} from './action/action.store';
import {PipelineResolver} from './pipeline/pipeline.resolver';
import {ApplicationResolver, ApplicationQueryParamResolver} from './application/application.resolver';
import {ProjectResolver} from './project/project.resolver';
import {ApplicationTemplateService} from './application/application.template.service';
import {ProjectAuditService} from './project/project.audit.service';
import {EnvironmentAuditService} from './environment/environment.audit.service';
import {ApplicationAuditService} from './application/application.audit.service';

@NgModule({})
export class ServicesModule {

    static forRoot(): ModuleWithProviders {
        return {
            ngModule: ServicesModule,
            providers: [
                ApplicationAuditService,
                ApplicationResolver,
                ApplicationQueryParamResolver,
                ActionService,
                ActionStore,
                ApplicationService,
                ApplicationWorkflowService,
                ApplicationPipelineService,
                ApplicationTemplateService,
                ApplicationStore,
                AuthentificationStore,
                CanActivateAuthRoute,
                EnvironmentAuditService,
                GroupService,
                ParameterService,
                PipelineResolver,
                PipelineService,
                PipelineStore,
                ProjectResolver,
                ProjectService,
                ProjectAuditService,
                ProjectStore,
                RepoManagerService,
                RequirementStore,
                RequirementService,
                UserService,
                VariableService,
                WarningStore,
                {
                    provide: Http,
                    useFactory: (httpFactory),
                    deps: [XHRBackend, RequestOptions, ToastService, AuthentificationStore, Router]
                }
            ]
        };
    }

    constructor (@Optional() @SkipSelf() parentModule: ServicesModule) {
        if (parentModule) {
            throw new Error(
                'ServicesModule is already loaded. Import it in the AppModule only');
        }
    }
}

export function httpFactory(backend: XHRBackend, defaultOptions: RequestOptions,
                            toast: ToastService, authStore: AuthentificationStore, router: Router) {
    return new HttpService(backend, defaultOptions, toast, authStore, router);
}

export {
    ApplicationAuditService,
    ActionStore,
    ApplicationResolver,
    ApplicationStore,
    ApplicationPipelineService,
    ApplicationWorkflowService,
    ApplicationTemplateService,
    AuthentificationStore,
    CanActivateAuthRoute,
    EnvironmentAuditService,
    GroupService,
    ParameterService,
    PipelineResolver,
    PipelineStore,
    ProjectResolver,
    ProjectStore,
    ProjectAuditService,
    RepoManagerService,
    RequirementStore,
    UserService,
    VariableService,
    WarningStore,
    Http
}
