import { HTTP_INTERCEPTORS } from '@angular/common/http';
import {
    ModuleWithProviders,
    NgModule,
    Optional,
    SkipSelf
} from '@angular/core';
import { AscodeService } from 'app/service/ascode/ascode.service';
import { ActionService } from './action/action.service';
import { ApplicationAuditService } from './application/application.audit.service';
import {
    ApplicationQueryParamResolver,
    ApplicationResolver
} from './application/application.resolver';
import { ApplicationService } from './application/application.service';
import { ApplicationStore } from './application/application.store';
import { ApplicationWorkflowService } from './application/application.workflow.service';
import { AuthenticationService } from './authentication/authentication.service';
import { ErrorInterceptor } from './authentication/error.interceptor';
import { LogoutInterceptor } from './authentication/logout.interceptor';
import { ProxyInterceptor } from './authentication/proxy.interceptor';
import { XSRFInterceptor } from './authentication/xsrf.interceptor';
import { BroadcastService } from './broadcast/broadcast.service';
import { BroadcastStore } from './broadcast/broadcast.store';
import { ConfigService } from './config/config.service';
import { DownloadService } from './download/download.service';
import { EnvironmentAuditService } from './environment/environment.audit.service';
import { EnvironmentService } from './environment/environment.service';
import { FeatureService } from './feature/feature.service';
import { GroupService } from './group/group.service';
import { HelpService } from './help/help.service';
import { HelpersService } from './helpers/helpers.service';
import { HookService } from './hook/hook.service';
import { ImportAsCodeService } from './import-as-code/import.service';
import { IntegrationService } from './integration/integration.service';
import { KeyService } from './keys/keys.service';
import { MonitoringService } from './monitoring/monitoring.service';
import { NavbarService } from './navbar/navbar.service';
import { NotificationService } from './notification/notification.service';
import { ParameterService } from './parameter/parameter.service';
import { PipelineCoreService } from './pipeline/pipeline.core.service';
import { PipelineService } from './pipeline/pipeline.service';
import { ProjectAuditService } from './project/project.audit.service';
import {
    ProjectForApplicationResolver,
    ProjectForWorkflowResolver,
    ProjectResolver
} from './project/project.resolver';
import { ProjectService } from './project/project.service';
import { ProjectStore } from './project/project.store';
import { QueueService } from './queue/queue.service';
import { RepoManagerService } from './repomanager/project.repomanager.service';
import { RequirementService } from './requirement/requirement.service';
import { RequirementStore } from './requirement/requirement.store';
import { RouterService } from './router/router.service';
import { ServiceService } from './service/service.service';
import { ThemeStore } from './theme/theme.store';
import { TimelineService } from './timeline/timeline.service';
import { TimelineStore } from './timeline/timeline.store';
import { UserService } from './user/user.service';
import { VariableService } from './variable/variable.service';
import { WorkerModelService } from './worker-model/worker-model.service';
import { WorkflowTemplateService } from './workflow-template/workflow-template.service';
import { WorkflowRunService } from './workflow/run/workflow.run.service';
import { WorkflowCoreService } from './workflow/workflow.core.service';
import { WorkflowService } from './workflow/workflow.service';
import { WorkflowStore } from './workflow/workflow.store';

@NgModule({})
export class ServicesModule {

    constructor(@Optional() @SkipSelf() parentModule: ServicesModule) {
        if (parentModule) {
            throw new Error(
                'ServicesModule is already loaded. Import it in the AppModule only');
        }
    }

    static forRoot(): ModuleWithProviders<ServicesModule> {
        return {
            ngModule: ServicesModule,
            providers: [
                ApplicationAuditService,
                ApplicationResolver,
                ApplicationQueryParamResolver,
                ActionService,
                ApplicationWorkflowService,
                ApplicationService,
                ApplicationStore,
                AscodeService,
                AuthenticationService,
                ConfigService,
                DownloadService,
                EnvironmentAuditService,
                EnvironmentService,
                FeatureService,
                GroupService,
                HelpService,
                HookService,
                HelpersService,
                ImportAsCodeService,
                BroadcastService,
                BroadcastStore,
                KeyService,
                ThemeStore,
                NavbarService,
                NotificationService,
                ParameterService,
                MonitoringService,
                PipelineCoreService,
                PipelineService,
                IntegrationService,
                ProjectResolver,
                ProjectForApplicationResolver,
                ProjectForWorkflowResolver,
                ProjectService,
                ProjectAuditService,
                ProjectStore,
                QueueService,
                RepoManagerService,
                RequirementStore,
                RequirementService,
                RouterService,
                ServiceService,
                TimelineService,
                TimelineStore,
                UserService,
                VariableService,
                WorkerModelService,
                WorkflowTemplateService,
                WorkflowService,
                WorkflowStore, WorkflowRunService, WorkflowCoreService,
                {
                    provide: HTTP_INTERCEPTORS,
                    useClass: ProxyInterceptor,
                    multi: true
                },
                {
                    provide: HTTP_INTERCEPTORS,
                    useClass: XSRFInterceptor,
                    multi: true
                },
                {
                    provide: HTTP_INTERCEPTORS,
                    useClass: LogoutInterceptor,
                    multi: true
                },
                {
                    provide: HTTP_INTERCEPTORS,
                    useClass: ErrorInterceptor,
                    multi: true
                }
            ]
        };
    }
}

export {
    ApplicationAuditService,
    ApplicationWorkflowService,
    ApplicationResolver,
    ApplicationStore,
    AscodeService,
    AuthenticationService,
    ConfigService,
    DownloadService,
    EnvironmentAuditService,
    GroupService,
    HelpService,
    HelpersService,
    HookService,
    ImportAsCodeService,
    BroadcastStore,
    KeyService,
    ThemeStore,
    ParameterService,
    MonitoringService,
    PipelineCoreService,
    IntegrationService,
    ProjectResolver,
    ProjectForApplicationResolver,
    ProjectForWorkflowResolver,
    ProjectStore,
    ProjectAuditService,
    QueueService,
    RepoManagerService,
    RequirementStore,
    RouterService,
    ServiceService,
    TimelineStore,
    UserService,
    VariableService,
    WorkerModelService,
    WorkflowTemplateService,
    WorkflowStore,
    WorkflowRunService,
    WorkflowCoreService
};

