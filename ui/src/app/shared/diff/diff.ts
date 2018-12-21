import { WorkflowTemplate } from '../../model/workflow-template.model';
import { Item } from './list/diff.list.component';

export function calculateWorkflowTemplateDiff(before: WorkflowTemplate, after: WorkflowTemplate): Array<Item> {
    let beforeTemplate: any;
    if (before) {
        beforeTemplate = {
            name: before.name,
            slug: before.slug,
            group_id: before.group_id,
            description: before.description,
            parameters: before.parameters
        }
    }

    let afterTemplate: any;
    if (after) {
        afterTemplate = {
            name: after.name,
            slug: after.slug,
            group_id: after.group_id,
            description: after.description,
            parameters: after.parameters
        }
    }

    let diffItems = [
        <Item>{
            translate: 'workflow_template',
            before: beforeTemplate ? JSON.stringify(beforeTemplate) : null,
            after: afterTemplate ? JSON.stringify(afterTemplate) : null,
            type: 'application/json'
        },
        <Item>{
            translate: 'workflow_template_diff_workflow',
            before: before ? atob(before.value) : null,
            after: after ? atob(after.value) : null,
            type: 'text/x-yaml'
        }
    ];

    let pipelinesLength = Math.max(before ? before.pipelines.length : 0, after ? after.pipelines.length : 0);
    for (let i = 0; i < pipelinesLength; i++) {
        diffItems.push(
            <Item>{
                translate: 'workflow_template_diff_pipeline',
                translateData: { number: pipelinesLength > 1 ? i : '' },
                before: before && before.pipelines[i] ? atob(before.pipelines[i].value) : null,
                after: after && after.pipelines[i] ? atob(after.pipelines[i].value) : null,
                type: 'text/x-yaml'
            })
    }

    let applicationsLength = Math.max(before ? before.applications.length : 0, after ? after.applications.length : 0);
    for (let i = 0; i < applicationsLength; i++) {
        diffItems.push(
            <Item>{
                translate: 'workflow_template_diff_application',
                translateData: { number: applicationsLength > 1 ? i : '' },
                before: before && before.applications[i] ? atob(before.applications[i].value) : null,
                after: after && after.applications[i] ? atob(after.applications[i].value) : null,
                type: 'text/x-yaml'
            })
    }

    let environmentsLength = Math.max(before ? before.environments.length : 0, after ? after.environments.length : 0);
    for (let i = 0; i < environmentsLength; i++) {
        diffItems.push(
            <Item>{
                translate: 'workflow_template_diff_environment',
                translateData: { number: environmentsLength > 1 ? i : '' },
                before: before && before.environments[i] ? atob(before.environments[i].value) : null,
                after: after && after.environments[i] ? atob(after.environments[i].value) : null,
                type: 'text/x-yaml'
            })
    }

    return diffItems;
}
