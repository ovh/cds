import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { NgxsReduxDevtoolsPluginModule } from '@ngxs/devtools-plugin';
import { NgxsLoggerPluginModule } from '@ngxs/logger-plugin';
import { NgxsModule } from '@ngxs/store';
import { ApplicationsState } from 'app/store/applications.state';
import { environment as env } from '../../environments/environment';


@NgModule({
    imports: [
        CommonModule,
        NgxsLoggerPluginModule.forRoot({ logger: console, collapsed: false }),
        NgxsReduxDevtoolsPluginModule.forRoot({ disabled: env.production }),
        NgxsModule.forRoot([ApplicationsState], { developmentMode: !env.production })
    ],
    exports: [
        NgxsLoggerPluginModule,
        NgxsReduxDevtoolsPluginModule,
        NgxsModule
    ]
})
export class NgxsStoreModule { }
