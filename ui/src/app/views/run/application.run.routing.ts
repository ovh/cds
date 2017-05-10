import {ModuleWithProviders} from '@angular/core';
import {Routes, RouterModule} from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {ApplicationPipelineBuildComponent} from './pipeline.build.component';
import {PipelineResolver} from '../../service/pipeline/pipeline.resolver';
import {ApplicationResolver} from '../../service/application/application.resolver';
import {ProjectResolver} from '../../service/project/project.resolver';

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
