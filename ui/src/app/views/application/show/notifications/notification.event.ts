import {Notification} from '../../../../model/notification.model';

export class NotificationEvent {
    notifications: Array<Notification>;
    type: string;

    constructor(t: string, n: Array<Notification>) {
        this.notifications = n;
        this.type = t;
    }
}
