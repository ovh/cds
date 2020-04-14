import { WorkflowTemplate } from 'app/model/workflow-template.model';
import { Base64 } from 'app/shared/base64.utils';
import { Item } from 'app/shared/diff/list/diff.list.component';


export function calculateWorkflowTemplateDiff(before: WorkflowTemplate, after: WorkflowTemplate): Array<Item> {
    let beforeTemplate: any;
    if (before && before.slug) {
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
            before: before ? Base64.b64DecodeUnicode(before.value) : null,
            after: after ? Base64.b64DecodeUnicode(after.value) : null,
            type: 'text/x-yaml'
        }
    ];

    let pipelinesLength = Math.max(before && before.pipelines ? before.pipelines.length : 0,
        after && after.pipelines ? after.pipelines.length : 0);
    for (let i = 0; i < pipelinesLength; i++) {
        diffItems.push(
            <Item>{
                translate: 'workflow_template_diff_pipeline',
                translateData: { number: pipelinesLength > 1 ? i : '' },
                before: before && before.pipelines && before.pipelines[i] ? Base64.b64DecodeUnicode(before.pipelines[i].value) : null,
                after: after && after.pipelines && after.pipelines[i] ? Base64.b64DecodeUnicode(after.pipelines[i].value) : null,
                type: 'text/x-yaml'
            })
    }

    let applicationsLength = Math.max(
        before && before.applications ? before.applications && before.applications.length : 0,
        after && after.applications ? after.applications && after.applications.length : 0
    );
    for (let i = 0; i < applicationsLength; i++) {
        diffItems.push(
            <Item>{
                translate: 'workflow_template_diff_application',
                translateData: { number: applicationsLength > 1 ? i : '' },
                before: before && before.applications && before.applications[i] ?
                    Base64.b64DecodeUnicode(before.applications[i].value) : null,
                after: after && after.applications && after.applications[i] ?
                    Base64.b64DecodeUnicode(after.applications[i].value) : null,
                type: 'text/x-yaml'
            })
    }

    let environmentsLength = Math.max(
        before && before.environments ? before.environments && before.environments.length : 0,
        after && after.environments ? after.environments && after.environments.length : 0
    );
    for (let i = 0; i < environmentsLength; i++) {
        diffItems.push(
            <Item>{
                translate: 'workflow_template_diff_environment',
                translateData: { number: environmentsLength > 1 ? i : '' },
                before: before && before.environments && before.environments[i] ?
                    Base64.b64DecodeUnicode(before.environments[i].value) : null,
                after: after && after.environments && after.environments[i] ?
                    Base64.b64DecodeUnicode(after.environments[i].value) : null,
                type: 'text/x-yaml'
            })
    }

    return diffItems;
}
