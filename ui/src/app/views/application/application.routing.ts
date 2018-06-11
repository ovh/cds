import {ModuleWithProviders} from '@angular/core';
import {Routes, RouterModule} from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {ApplicationShowComponent} from './show/application.component';
import {ApplicationAddComponent} from './add/application.add.component';
import {ProjectForApplicationResolver, ProjectForWorkflowResolver} from '../../service/project/project.resolver';

const applicationRoutes: Routes = [
    {
        path: '',
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        children: [
            { path: '', component: ApplicationAddComponent ,
                data: { title: 'Add • Application' },
                resolve: {
                    project: ProjectForWorkflowResolver
                }
            },
            { path: ':appName',
                component: ApplicationShowComponent,
                data: { title: '{appName} • Application' },
                resolve: {
                    project: ProjectForApplicationResolver
                }
            },
            {
                path: ':appName/pipeline/:pipName/build',
                loadChildren: 'app/views/run/application.run.module#ApplicationRunModule',
                data: { title: 'Pipeline {pipName} • Application {appName}' }
            }
        ]
    }
];


export const applicationRouting: ModuleWithProviders = RouterModule.forChild(applicationRoutes);
