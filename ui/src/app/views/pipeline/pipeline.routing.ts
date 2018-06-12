import { ModuleWithProviders } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {PipelineShowComponent} from './show/pipeline.show.component';
import {ProjectResolver} from '../../service/project/project.resolver';
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
                    project: ProjectResolver
                },
                data: { title: 'Add • Pipeline' }
            },
            { path: ':pipName',
                component: PipelineShowComponent,
                data: { title: '{pipName} • Pipeline' },
                resolve: {
                    project: ProjectResolver,
                    application: ApplicationQueryParamResolver
                }
            }
        ]
    }
];

export const pipelineRouting: ModuleWithProviders = RouterModule.forChild(pipelineRoutes);
