import {Component, Input, Output, EventEmitter, ViewChild} from '@angular/core';
import {Table} from '../../../../../shared/table/table';
import {Project} from '../../../../../model/project.model';
import {Application} from '../../../../../model/application.model';
import {Notification} from '../../../../../model/notification.model';
import {NotificationEvent} from '../notification.event';
import {ApplicationNotificationFormModalComponent} from '../form/notification.form.component';

declare var _: any;

@Component({
    selector: 'app-notification-list',
    templateUrl: './notification.list.html',
    styleUrls: ['./notification.list.scss']
})
export class ApplicationNotificationListComponent extends Table {

    @Input() notifications: Array<Notification>;
    @Input() edit = false;
    @Input() project: Project;
    @Input() application: Application;
    @Input() loading = false;

    @Output() event = new EventEmitter<NotificationEvent>();

    @ViewChild('editForm')
    editNotifModal: ApplicationNotificationFormModalComponent;

    selectedNotification: Notification;

    constructor() {
        super();
    }

    getData(): any[] {
        return this.notifications;
    }

    sendEvent(type: string, n: Notification) {
        n.updating = true;
        let notifs = new Array<Notification>();
        notifs.push(n);
        this.send(new NotificationEvent(type, notifs));

    }

    send(ne: NotificationEvent) {
        this.event.emit(ne);
    }

    editNotification(n: Notification, key: string) {
        this.selectedNotification = new Notification();
        this.selectedNotification.pipeline = _.cloneDeep(n.pipeline);
        this.selectedNotification.environment = _.cloneDeep(n.environment);
        this.selectedNotification.application_pipeline_id = n.application_pipeline_id;
        this.selectedNotification.notifications[key] = _.cloneDeep(n.notifications[key]);
        this.editNotifModal.show({autofocus: false, closable: false, observeChanges: false});
    }

    close(): void {
        this.editNotifModal.close();
    }
}
