import { ModuleWithProviders } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';
import {CanActivateAuthRoute} from '../../../service/auth/authenRouteActivate';
import {PipelineListComponent} from './list/pipeline.list.component';
import {PipelineShowComponent} from './show/pipeline.show.component';
import {ProjectResolver} from '../../../service/project/project.resolver';
import {ApplicationQueryParamResolver} from '../../../service/application/application.resolver';

const pipelineRoutes: Routes = [
    {
        path: '',
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        children: [
            { path: '', component: PipelineListComponent },
            { path: ':pipName',
                component: PipelineShowComponent,
                resolve: {
                    project: ProjectResolver,
                    application: ApplicationQueryParamResolver,
                }
            }
        ]
    }
];

export const pipelineRouting: ModuleWithProviders = RouterModule.forChild(pipelineRoutes);
