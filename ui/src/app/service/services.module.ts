import {NgModule, ModuleWithProviders, SkipSelf, Optional} from '@angular/core';
import {ProjectService} from './project/project.service';
import {ProjectStore} from './project/project.store';
import {Http} from '@angular/http';
import {AuthentificationStore} from './auth/authentification.store';
import {UserService} from './user/user.service';
import {CanActivateAuthRoute} from './auth/authenRouteActivate';
import {CanActivateAuthAdminRoute} from './auth/authenAdminRouteActivate';
import {WarningStore} from './warning/warning.store';
import {PipelineStore} from './pipeline/pipeline.store';
import {PipelineService} from './pipeline/pipeline.service';
import {ApplicationService} from './application/application.service';
import {ApplicationStore} from './application/application.store';
import {ApplicationPipelineService} from './application/pipeline/application.pipeline.service';
import {VariableService} from './variable/variable.service';
import {GroupService} from './group/group.service';
import {RepoManagerService} from './repomanager/project.repomanager.service';
import {ApplicationWorkflowService} from './application/application.workflow.service';
import {RequirementService} from './worker-model/requirement/requirement.service';
import {RequirementStore} from './worker-model/requirement/requirement.store';
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
import {WorkerModelService} from './worker-model/worker-model.service';
import {LanguageStore} from './language/language.store';
import {NotificationService} from './notification/notification.service';
import {WorkflowService} from './workflow/workflow.service';
import {WorkflowStore} from './workflow/workflow.store';
import {WorkflowRunService} from './workflow/run/workflow.run.service';
import {RouterService} from './router/router.service';
import {WarningService} from './warning/warning.service';
import {LastUpdateService} from './sse/lastupdate.sservice';
import {HTTP_INTERCEPTORS} from '@angular/common/http';
import {AuthentificationInterceptor} from './auth.interceptor.service';
import {LogoutInterceptor} from './logout.interceptor.service';
import {HookService} from './hook/hook.service';

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
                CanActivateAuthAdminRoute,
                EnvironmentAuditService,
                GroupService,
                HookService,
                LanguageStore,
                LastUpdateService,
                NotificationService,
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
                RouterService,
                UserService,
                VariableService,
                WarningService,
                WarningStore,
                WorkerModelService,
                WorkflowService, WorkflowStore, WorkflowRunService,
                {
                    provide: HTTP_INTERCEPTORS,
                    useClass: AuthentificationInterceptor,
                    multi: true
                },
                {
                    provide: HTTP_INTERCEPTORS,
                    useClass: LogoutInterceptor,
                    multi: true
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
    CanActivateAuthAdminRoute,
    EnvironmentAuditService,
    GroupService,
    HookService,
    LanguageStore,
    LastUpdateService,
    ParameterService,
    PipelineResolver,
    PipelineStore,
    ProjectResolver,
    ProjectStore,
    ProjectAuditService,
    RepoManagerService,
    RequirementStore,
    RouterService,
    UserService,
    VariableService,
    WarningStore,
    WarningService,
    WorkerModelService,
    WorkflowStore,
    WorkflowRunService,
    Http
}
