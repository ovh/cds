import {WorkflowNodeHookConfigValue} from './workflow.model';

export class WorkflowHookModel {
    id: number;
    name: string;
    type: string;
    author: string;
    description: string;
    identifier: string;
    image: string;
    command: string;
    icon: string;
    default_config: Map<string, WorkflowNodeHookConfigValue>;
    disabled: boolean;
}
