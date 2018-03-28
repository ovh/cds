import {Pipeline} from './pipeline.model';
import {Environment} from './environment.model';

export const notificationTypes = ['jabber', 'email'];
export const notificationOnSuccess = ['always', 'change', 'never'];
export const notificationOnFailure = ['always', 'change', 'never'];

export class Notification {
    application_pipeline_id: number;
    pipeline: Pipeline;
    environment: Environment;
    notifications: any;

    // UI attribute
    updating = false;

    constructor() {
        this.notifications = {};
    }
}

export class UserNotificationSettings {
    on_success: string;
    on_failure: string;
    on_start: boolean;
    send_to_groups: boolean;
    send_to_author: boolean;
    recipients: Array<string>;
    template: UserNotificationTemplate;

    constructor() {
        this.on_success = notificationOnSuccess[1];
        this.on_failure = notificationOnFailure[0];
        this.on_start = false;
        this.send_to_author = true;
        this.send_to_groups = false;
        this.recipients = [];
        this.template = new UserNotificationTemplate();
    }
}

export class UserNotificationTemplate {
    subject: string;
    body: string;

    constructor() {
        this.subject = '{{.cds.project}}/{{.cds.application}} {{.cds.pipeline}} {{.cds.environment}}#{{.cds.version}} {{.cds.status}}';
        this.body = 'Project : {{.cds.project}}\n' +
                'Application : {{.cds.application}}\n' +
                'Pipeline : {{.cds.pipeline}}/{{.cds.environment}}#{{.cds.version}}\n' +
                'Status : {{.cds.status}}\n' +
                'Details : {{.cds.buildURL}}\n' +
                'Triggered by : {{.cds.triggered_by.username}}\n' +
                'Branch : {{.git.branch}}';
    }
}
