import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { NgxsReduxDevtoolsPluginModule } from '@ngxs/devtools-plugin';
import { NgxsLoggerPluginModule } from '@ngxs/logger-plugin';
import { NgxsModule } from '@ngxs/store';
import { ApplicationsState } from 'app/store/applications.state';
import { environment as env } from '../../environments/environment';
import { ProjectState } from './project.state';


@NgModule({
    imports: [
        CommonModule,
        NgxsLoggerPluginModule.forRoot({ logger: console, collapsed: false, disabled: env.production }),
        NgxsReduxDevtoolsPluginModule.forRoot({ disabled: env.production }),
        NgxsModule.forRoot([ProjectState, ApplicationsState], { developmentMode: !env.production })
    ],
    exports: [
        NgxsLoggerPluginModule,
        NgxsReduxDevtoolsPluginModule,
        NgxsModule
    ]
})
export class NgxsStoreModule { }
