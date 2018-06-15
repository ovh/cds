import {ModuleWithProviders} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {ApplicationResolver} from '../../service/application/application.resolver';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {PipelineResolver} from '../../service/pipeline/pipeline.resolver';
import {ProjectResolver} from '../../service/project/project.resolver';
import {ApplicationPipelineBuildComponent} from './pipeline.build.component';

const applicationRunRoutes: Routes = [
    {
        path: '',
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        children: [
            { path: ':buildNumber',
                component: ApplicationPipelineBuildComponent,
                resolve: {
                    pipeline: PipelineResolver,
                    application: ApplicationResolver,
                    project: ProjectResolver
                }
            }
        ]
    }
];


export const applicationRunRouting: ModuleWithProviders = RouterModule.forChild(applicationRunRoutes);
