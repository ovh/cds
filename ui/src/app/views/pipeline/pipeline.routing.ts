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
                data: { title: 'CDS - Add a pipeline' }
            },
            { path: ':pipName',
                component: PipelineShowComponent,
                data: { title: 'CDS - Pipeline {pipName}' },
                resolve: {
                    project: ProjectResolver,
                    application: ApplicationQueryParamResolver
                }
            }
        ]
    }
];

export const pipelineRouting: ModuleWithProviders = RouterModule.forChild(pipelineRoutes);
