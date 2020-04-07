import { WorkflowNodeHookConfigValue } from './workflow.model';

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

export enum HookStatus {
    DONE = 'DONE',
    DOING = 'DOING',
    FAIL = 'FAIL',
    SCHEDULED = 'SCHEDULED'
}

export class WorkflowHookTask {
    uuid: string;
    stopped: boolean;
    config: Map<string, WorkflowNodeHookConfigValue>;
    type: string;
    executions: TaskExecution[];
    nb_executions_total: number;
    nb_executions_todo: number;
}

export class TaskExecution {
    uuid: string;
    type: string;
    timestamp: number;
    nb_errors: number;
    last_error: string;
    processing_timestamp: number;
    workflow_run: number;
    config: Map<string, WorkflowNodeHookConfigValue>;
    webhook: Webhook;
    rabbitmq: RabbitMQ;
    gerrit: GerritExecution;
    kafka: Kafka;
    scheduled_task?: any;
    status: HookStatus;
}

export class GerritExecution {
    message: string;
}

export class Webhook {
    reques_url: string;
    request_body: string;
    request_header: Map<string, string[]>;
}

export class RabbitMQ {
    message: string;
}

export class Kafka {
    message: string;
}

