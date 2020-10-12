import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { NgxsReduxDevtoolsPluginModule } from '@ngxs/devtools-plugin';
import { NgxsLoggerPluginModule } from '@ngxs/logger-plugin';
import { NgxsModule } from '@ngxs/store';
import { ServicesModule } from 'app/service/services.module';
import { SharedModule } from 'app/shared/shared.module';
import { ApplicationsState } from 'app/store/applications.state';
import { CDSState } from 'app/store/cds.state';
import { EnvironmentState } from 'app/store/environment.state';
import { PipelinesState } from 'app/store/pipelines.state';
import { environment as env } from '../../environments/environment';
import { AuthenticationState } from './authentication.state';
import { EventState } from './event.state';
import { FeatureState } from './feature.state';
import { HelpState } from './help.state';
import { ProjectState } from './project.state';
import { QueueState } from './queue.state';
import { WorkflowState } from './workflow.state';

@NgModule({
    imports: [
        CommonModule,
        ServicesModule,
        SharedModule,
        NgxsLoggerPluginModule.forRoot({ logger: console, collapsed: false, disabled: env.production }),
        NgxsReduxDevtoolsPluginModule.forRoot({ disabled: env.production }),
        NgxsModule.forRoot([
            AuthenticationState,
            ApplicationsState,
            CDSState,
            EnvironmentState,
            EventState,
            HelpState,
            PipelinesState,
            ProjectState,
            QueueState,
            WorkflowState,
            FeatureState
        ], { developmentMode: !env.production })
    ],
    exports: [
        NgxsLoggerPluginModule,
        NgxsReduxDevtoolsPluginModule,
        NgxsModule
    ]
})
export class NgxsStoreModule { }
