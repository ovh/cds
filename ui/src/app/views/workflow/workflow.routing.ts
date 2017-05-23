import {ModuleWithProviders} from '@angular/core';
import {Routes, RouterModule} from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {ProjectResolver} from '../../service/project/project.resolver';
import {WorkflowAddComponent} from './add/workflow.add.component';
import {WorkflowShowComponent} from './show/workflow.component';

const workflowRoutes: Routes = [
    {
        path: '',
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        children: [
            { path: '', component: WorkflowAddComponent,
                resolve: {
                    project: ProjectResolver
                }
            },
            { path: ':workflowName',
                component: WorkflowShowComponent,
                resolve: {
                    project: ProjectResolver
                }
            }
        ]
    }
];


export const workflowRouting: ModuleWithProviders = RouterModule.forChild(workflowRoutes);
