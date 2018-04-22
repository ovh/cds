import {NgModule, ModuleWithProviders, SkipSelf, Optional} from '@angular/core';
import {ProjectService} from './project/project.service';
import {ProjectStore} from './project/project.store';
import {Http} from '@angular/http';
import {AuthentificationStore} from './auth/authentification.store';
import {UserService} from './user/user.service';
import {CanActivateAuthRoute} from './auth/authenRouteActivate';
import {CanActivateAuthAdminRoute} from './auth/authenAdminRouteActivate';
import {PipelineStore} from './pipeline/pipeline.store';
import {PipelineService} from './pipeline/pipeline.service';
import {ApplicationService} from './application/application.service';
import {ApplicationStore} from './application/application.store';
import {ApplicationPipelineService} from './application/pipeline/application.pipeline.service';
import {VariableService} from './variable/variable.service';
import {GroupService} from './group/group.service';
import {RepoManagerService} from './repomanager/project.repomanager.service';
import {ApplicationWorkflowService} from './application/application.workflow.service';
import {RequirementService} from './requirement/requirement.service';
import {RequirementStore} from './requirement/requirement.store';
import {ParameterService} from './parameter/parameter.service';
import {MonitoringService} from './monitoring/monitoring.service';
import {ActionService} from './action/action.service';
import {ActionStore} from './action/action.store';
import {PipelineResolver} from './pipeline/pipeline.resolver';
import {ApplicationResolver, ApplicationQueryParamResolver} from './application/application.resolver';
import {
    ProjectResolver,
    ProjectForApplicationResolver,
    ProjectForWorkflowResolver
} from './project/project.resolver';
import {ProjectAuditService} from './project/project.audit.service';
import {EnvironmentAuditService} from './environment/environment.audit.service';
import {ApplicationAuditService} from './application/application.audit.service';
import {WorkerModelService} from './worker-model/worker-model.service';
import {LanguageStore} from './language/language.store';
import {NotificationService} from './notification/notification.service';
import {WorkflowService} from './workflow/workflow.service';
import {WorkflowStore} from './workflow/workflow.store';
import {WorkflowRunService} from './workflow/run/workflow.run.service';
import {WorkflowCoreService} from './workflow/workflow.core.service';
import {RouterService} from './router/router.service';
import {LastUpdateService} from './sse/lastupdate.sservice';
import {HTTP_INTERCEPTORS} from '@angular/common/http';
import {AuthentificationInterceptor} from './auth.interceptor.service';
import {LogoutInterceptor} from './logout.interceptor.service';
import {HookService} from './hook/hook.service';
import {PipelineAuditService} from './pipeline/pipeline.audit.service';
import {EnvironmentService} from './environment/environment.service';
import {ApplicationMigrateService} from './application/application.migration.service';
import {NavbarService} from './navbar/navbar.service';
import {DownloadService} from './download/download.service';
import {KeyService} from './keys/keys.service';
import {PlatformService} from './platform/platform.service';
import {ImportAsCodeService} from './import-as-code/import.service';
import {BroadcastService} from './broadcast/broadcast.service';

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
                ApplicationMigrateService,
                ApplicationStore,
                AuthentificationStore,
                DownloadService,
                CanActivateAuthRoute,
                CanActivateAuthAdminRoute,
                EnvironmentAuditService,
                EnvironmentService,
                GroupService,
                HookService,
                ImportAsCodeService,
                BroadcastService,
                KeyService,
                LanguageStore,
                LastUpdateService,
                NavbarService,
                NotificationService,
                ParameterService,
                MonitoringService,
                PipelineResolver,
                PipelineService,
                PipelineAuditService,
                PipelineStore,
                PlatformService,
                ProjectResolver,
                ProjectForApplicationResolver,
                ProjectForWorkflowResolver,
                ProjectService,
                ProjectAuditService,
                ProjectStore,
                RepoManagerService,
                RequirementStore,
                RequirementService,
                RouterService,
                UserService,
                VariableService,
                WorkerModelService,
                WorkflowService, WorkflowStore, WorkflowRunService, WorkflowCoreService,
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
    ApplicationMigrateService,
    AuthentificationStore,
    CanActivateAuthRoute,
    CanActivateAuthAdminRoute,
    DownloadService,
    EnvironmentAuditService,
    GroupService,
    HookService,
    ImportAsCodeService,
    BroadcastService,
    KeyService,
    LanguageStore,
    LastUpdateService,
    ParameterService,
    MonitoringService,
    PipelineResolver,
    PipelineStore,
    PipelineAuditService,
    PlatformService,
    ProjectResolver,
    ProjectForApplicationResolver,
    ProjectForWorkflowResolver,
    ProjectStore,
    ProjectAuditService,
    RepoManagerService,
    RequirementStore,
    RouterService,
    UserService,
    VariableService,
    WorkerModelService,
    WorkflowStore,
    WorkflowRunService,
    WorkflowCoreService,
    Http
}
