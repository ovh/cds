import {ModuleWithProviders} from '@angular/core';
import {Routes, RouterModule} from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {ApplicationShowComponent} from './show/application.component';
import {ApplicationAddComponent} from './add/application.add.component';
import {ProjectForApplicationResolver} from '../../service/project/project.resolver';

const applicationRoutes: Routes = [
    {
        path: '',
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        children: [
            { path: '', component: ApplicationAddComponent ,
                resolve: {
                    project: ProjectForApplicationResolver
                }
            },
            { path: ':appName',
                component: ApplicationShowComponent,
                resolve: {
                    project: ProjectForApplicationResolver
                }
            },
            {
                path: ':appName/pipeline/:pipName/build',
                loadChildren: 'app/views/run/application.run.module#ApplicationRunModule'
            }
        ]
    }
];


export const applicationRouting: ModuleWithProviders = RouterModule.forChild(applicationRoutes);
