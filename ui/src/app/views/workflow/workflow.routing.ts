import {ModuleWithProviders} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {ProjectResolver} from '../../service/project/project.resolver';
import {WorkflowAddComponent} from './add/workflow.add.component';
import {WorkflowShowComponent} from './show/workflow.component';
import {WorkflowRunComponent} from './run/workflow.run.component';
import {WorkflowNodeRunComponent} from './run/node/workflow.run.node.component';
import {WorkflowComponent} from './workflow.compoment';

const workflowRoutes: Routes = [
    {
        path: '',
        component: WorkflowAddComponent,
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        resolve: {
            project: ProjectResolver
        }
    },
    {
        path: ':workflowName',
        component: WorkflowComponent,
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        resolve: {
            project: ProjectResolver
        },
        children: [

            {
                path: '', component: WorkflowShowComponent,
                resolve: {
                    project: ProjectResolver
                }
            },
            {
                path: 'run/:number', component: WorkflowRunComponent,
                resolve: {
                    project: ProjectResolver
                }
            },
            {
                path: 'run/:number/node/:nodeId', component: WorkflowNodeRunComponent,
                resolve: {
                    project: ProjectResolver
                }
            }
        ]
    }
];


export const workflowRouting: ModuleWithProviders = RouterModule.forChild(workflowRoutes);
