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
    ProjectResolver, Projectv2Resolver
} from './project/project.resolver';
import { ProjectService } from './project/project.service';
import { ProjectStore } from './project/project.store';
import { QueueService } from './queue/queue.service';
import { RepoManagerService } from './repomanager/project.repomanager.service';
import { RequirementService } from './requirement/requirement.service';
import { RequirementStore } from './requirement/requirement.store';
import { RouterService } from './router/router.service';
import { ServiceService } from './service/service.service';
import { UserService } from './user/user.service';
import { VariableService } from './variable/variable.service';
import { WorkerModelService } from './worker-model/worker-model.service';
import { WorkflowTemplateService } from './workflow-template/workflow-template.service';
import { WorkflowRunService } from './workflow/run/workflow.run.service';
import { WorkflowCoreService } from './workflow/workflow.core.service';
import { WorkflowService } from './workflow/workflow.service';
import { WorkflowStore } from './workflow/workflow.store';
import { AnalysisService } from "./analysis/analysis.service";
import { LinkService } from "./link/link.service";
import { EntityService } from "./entity/entity.service";
import { ActionAsCodeService } from "./action/actionAscode.service";
import { PluginService } from "./plugin.service";
import { V2WorkflowRunService } from "./workflowv2/workflow.service";

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
                AnalysisService,
                ApplicationAuditService,
                ApplicationResolver,
                ApplicationQueryParamResolver,
                ActionService,
                ActionAsCodeService,
                ApplicationWorkflowService,
                ApplicationService,
                ApplicationStore,
                AscodeService,
                AuthenticationService,
                ConfigService,
                DownloadService,
                LinkService,
                EntityService,
                EnvironmentAuditService,
                EnvironmentService,
                FeatureService,
                GroupService,
                HelpService,
                HookService,
                HelpersService,
                ImportAsCodeService,
                KeyService,
                NavbarService,
                NotificationService,
                ParameterService,
                MonitoringService,
                PipelineCoreService,
                PipelineService,
                IntegrationService,
                PluginService,
                Projectv2Resolver,
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
                },
                V2WorkflowRunService
            ]
        };
    }
}

export {
    ActionAsCodeService,
    ApplicationAuditService,
    ApplicationWorkflowService,
    ApplicationResolver,
    ApplicationStore,
    AscodeService,
    AuthenticationService,
    ConfigService,
    DownloadService,
    EntityService,
    LinkService,
    EnvironmentAuditService,
    GroupService,
    HelpService,
    HelpersService,
    HookService,
    ImportAsCodeService,
    KeyService,
    ParameterService,
    MonitoringService,
    PipelineCoreService,
    IntegrationService,
    PluginService,
    ProjectResolver,
    Projectv2Resolver,
    ProjectForApplicationResolver,
    ProjectForWorkflowResolver,
    ProjectStore,
    ProjectAuditService,
    QueueService,
    RepoManagerService,
    RequirementStore,
    RouterService,
    ServiceService,
    UserService,
    VariableService,
    WorkerModelService,
    WorkflowTemplateService,
    WorkflowStore,
    WorkflowRunService,
    WorkflowCoreService,
    V2WorkflowRunService
};

