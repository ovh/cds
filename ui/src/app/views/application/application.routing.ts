import {ModuleWithProviders} from '@angular/core';
import {Routes, RouterModule} from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {ApplicationShowComponent} from './show/application.component';
import {ApplicationAddComponent} from './add/application.add.component';
import {ProjectResolver} from '../../service/project/project.resolver';

const applicationRoutes: Routes = [
    {
        path: '',
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        children: [
            { path: '', component: ApplicationAddComponent ,
                resolve: {
                    project: ProjectResolver
                }
            },
            { path: ':appName',
                component: ApplicationShowComponent,
                resolve: {
                    project: ProjectResolver
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
