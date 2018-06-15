import {ModuleWithProviders} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {ProjectForWorkflowResolver, ProjectResolver} from '../../service/project/project.resolver';
import {WorkflowAddComponent} from './add/workflow.add.component';
import {WorkflowNodeRunComponent} from './run/node/workflow.run.node.component';
import {WorkflowRunComponent} from './run/workflow.run.component';
import {WorkflowShowComponent} from './show/workflow.component';
import {WorkflowComponent} from './workflow.component';

const workflowRoutes: Routes = [
    {
        path: '',
        component: WorkflowAddComponent,
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        resolve: {
            project: ProjectForWorkflowResolver
        },
        data: {
          title: 'Add • Workflow'
        },
    },
    {
        path: ':workflowName',
        component: WorkflowComponent,
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        data: {
          title: '{workflowName} • Workflow'
        },
        resolve: {
            project: ProjectForWorkflowResolver
        },
        children: [
            {
                path: '', component: WorkflowShowComponent,
                resolve: {
                    project: ProjectForWorkflowResolver
                }
            },
            {
                path: 'run/:number', component: WorkflowRunComponent,
                resolve: {
                    project: ProjectResolver
                },
                data: {
                  title: '#{number} • {workflowName}'
                },
            },
            {
                path: 'run/:number/node/:nodeId', component: WorkflowNodeRunComponent,
                resolve: {
                    project: ProjectForWorkflowResolver
                },
                data: {
                  title: 'Pipeline {name} • #{number} • {workflowName}'
                },
            }
        ]
    }
];


export const workflowRouting: ModuleWithProviders = RouterModule.forChild(workflowRoutes);
