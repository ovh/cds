import {Pipeline} from './pipeline.model';
import {Environment} from './environment.model';

export interface Notification {
    application_pipeline_id: number;
    pipeline: Pipeline;
    environment: Environment;
};

export enum UserNotificationSettingsType {
    'jabber',
    'email',
    'tat'
}
