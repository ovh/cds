import {HTTP_INTERCEPTORS} from '@angular/common/http';
import {ModuleWithProviders, NgModule, Optional, SkipSelf} from '@angular/core';
import {ActionService} from './action/action.service';
import {ActionStore} from './action/action.store';
import {ApplicationAuditService} from './application/application.audit.service';
import {ApplicationMigrateService} from './application/application.migration.service';
import {ApplicationQueryParamResolver, ApplicationResolver} from './application/application.resolver';
import {ApplicationService} from './application/application.service';
import {ApplicationStore} from './application/application.store';
import {ApplicationWorkflowService} from './application/application.workflow.service';
import {ApplicationPipelineService} from './application/pipeline/application.pipeline.service';
import {AuthentificationInterceptor} from './auth.interceptor.service';
import {CanActivateAuthAdminRoute} from './auth/authenAdminRouteActivate';
import {CanActivateAuthRoute} from './auth/authenRouteActivate';
import {AuthentificationStore} from './auth/authentification.store';
import {BroadcastService} from './broadcast/broadcast.service';
import {BroadcastStore} from './broadcast/broadcast.store';
import {DownloadService} from './download/download.service';
import {EnvironmentAuditService} from './environment/environment.audit.service';
import {EnvironmentService} from './environment/environment.service';
import {GroupService} from './group/group.service';
import {HookService} from './hook/hook.service';
import {ImportAsCodeService} from './import-as-code/import.service';
import {KeyService} from './keys/keys.service';
import {LanguageStore} from './language/language.store';
import {LogoutInterceptor} from './logout.interceptor.service';
import {MonitoringService} from './monitoring/monitoring.service';
import {NavbarService} from './navbar/navbar.service';
import {NotificationService} from './notification/notification.service';
import {ParameterService} from './parameter/parameter.service';
import {PipelineAuditService} from './pipeline/pipeline.audit.service';
import {PipelineCoreService} from './pipeline/pipeline.core.service';
import {PipelineResolver} from './pipeline/pipeline.resolver';
import {PipelineService} from './pipeline/pipeline.service';
import {PipelineStore} from './pipeline/pipeline.store';
import {PlatformService} from './platform/platform.service';
import {ProjectAuditService} from './project/project.audit.service';
import {
    ProjectForApplicationResolver,
    ProjectForWorkflowResolver,
    ProjectResolver
} from './project/project.resolver';
import {ProjectService} from './project/project.service';
import {ProjectStore} from './project/project.store';
import {RepoManagerService} from './repomanager/project.repomanager.service';
import {RequirementService} from './requirement/requirement.service';
import {RequirementStore} from './requirement/requirement.store';
import {RouterService} from './router/router.service';
import {UserService} from './user/user.service';
import {VariableService} from './variable/variable.service';
import {WarningService} from './warning/warning.service';
import {WarningStore} from './warning/warning.store';
import {WorkerModelService} from './worker-model/worker-model.service';
import {WorkflowRunService} from './workflow/run/workflow.run.service';
import {WorkflowCoreService} from './workflow/workflow.core.service';
import {WorkflowEventStore} from './workflow/workflow.event.store';
import {WorkflowService} from './workflow/workflow.service';
import {WorkflowSidebarStore} from './workflow/workflow.sidebar.store';
import {WorkflowStore} from './workflow/workflow.store';

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
                BroadcastStore,
                KeyService,
                LanguageStore,
                NavbarService,
                NotificationService,
                ParameterService,
                MonitoringService,
                PipelineResolver,
                PipelineCoreService,
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
                WarningService,
                WarningStore,
                WorkerModelService,
                WorkflowEventStore,
                WorkflowSidebarStore,
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
    BroadcastStore,
    KeyService,
    LanguageStore,
    ParameterService,
    MonitoringService,
    PipelineResolver,
    PipelineCoreService,
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
    WarningStore,
    WorkerModelService,
    WorkflowStore,
    WorkflowRunService,
    WorkflowCoreService,
    WorkflowSidebarStore,
    WorkflowEventStore
}
