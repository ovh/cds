import {ModuleWithProviders} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {ProjectResolver} from '../../service/project/project.resolver';
import {WorkflowAddComponent} from './add/workflow.add.component';
import {WorkflowShowComponent} from './show/workflow.component';
import {WorkflowRunComponent} from './run/workflow.run.component';
import {WorkflowNodeRunComponent} from './run/node/workflow.run.node.component';

const workflowRoutes: Routes = [
    {
        path: '',
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        children: [
            {
                path: '', component: WorkflowAddComponent,
                resolve: {
                    project: ProjectResolver
                }
            },
            {
                path: ':workflowName', component: WorkflowShowComponent,
                resolve: {
                    project: ProjectResolver
                }
            },
            {
                path: ':workflowName/run/:number', component: WorkflowRunComponent,
                resolve: {
                    project: ProjectResolver
                }
            },
            {
                path: ':workflowName/run/:number/node/:nodeId/subnumber/:subnumber', component: WorkflowNodeRunComponent
            }
        ]
    }
];


export const workflowRouting: ModuleWithProviders = RouterModule.forChild(workflowRoutes);
