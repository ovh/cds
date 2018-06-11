import {ModuleWithProviders} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {CanActivateAuthRoute} from '../../service/auth/authenRouteActivate';
import {ProjectResolver, ProjectForWorkflowResolver} from '../../service/project/project.resolver';
import {WorkflowAddComponent} from './add/workflow.add.component';
import {WorkflowShowComponent} from './show/workflow.component';
import {WorkflowRunComponent} from './run/workflow.run.component';
import {WorkflowNodeRunComponent} from './run/node/workflow.run.node.component';
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
          title: 'CDS - Workflow - Add'
        },
    },
    {
        path: ':workflowName',
        component: WorkflowComponent,
        canActivate: [CanActivateAuthRoute],
        canActivateChild: [CanActivateAuthRoute],
        data: {
          title: 'CDS - Workflow {workflowName}'
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
                  title: 'CDS - Workflow {workflowName} - Run {number}'
                },
            },
            {
                path: 'run/:number/node/:nodeId', component: WorkflowNodeRunComponent,
                resolve: {
                    project: ProjectForWorkflowResolver
                },
                data: {
                  title: 'CDS - Workflow {workflowName} - Run {number} - Pipeline {name}'
                },
            }
        ]
    }
];


export const workflowRouting: ModuleWithProviders = RouterModule.forChild(workflowRoutes);
