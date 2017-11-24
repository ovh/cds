import { ModuleWithProviders } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {PipelineShowComponent} from './show/pipeline.show.component';
import {ProjectResolver, ProjectForPipelineCreateResolver} from '../../service/project/project.resolver';
import {ApplicationQueryParamResolver} from '../../service/application/application.resolver';
import {PipelineAddComponent} from './add/pipeline.add.component';

const pipelineRoutes: Routes = [
    {
        path: '',
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        children: [
            { path: '',
                component: PipelineAddComponent,
                resolve: {
                    project: ProjectForPipelineCreateResolver
                }
            },
            { path: ':pipName',
                component: PipelineShowComponent,
                resolve: {
                    project: ProjectResolver,
                    application: ApplicationQueryParamResolver
                }
            }
        ]
    }
];

export const pipelineRouting: ModuleWithProviders = RouterModule.forChild(pipelineRoutes);
