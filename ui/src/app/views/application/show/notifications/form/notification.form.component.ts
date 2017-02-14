import {Component, Input, ViewChild, Output, EventEmitter, AfterViewInit} from '@angular/core';
import {
    UserNotificationSettings, notificationTypes, notificationOnSuccess,
    notificationOnFailure, Notification
} from '../../../../../model/notification.model';
import {Project} from '../../../../../model/project.model';
import {Application, ApplicationPipeline} from '../../../../../model/application.model';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {FormControl} from '@angular/forms';
import {Environment} from '../../../../../model/environment.model';
import {NotificationEvent} from '../notification.event';
import {TranslateService} from 'ng2-translate';
import {DeleteButtonComponent} from '../../../../../shared/button/delete/delete.button';

@Component({
    selector: 'app-notification-form-modal',
    templateUrl: './notification.form.html',
    styleUrls: ['./notification.form.scss']
})
export class ApplicationNotificationFormModalComponent implements AfterViewInit {

    // Component output
    @Output() event = new EventEmitter<NotificationEvent>();

    // Component input
    @Input() project: Project;
    @Input() application: Application;
    @Input() loading = false;

    // Form data
    needEnv = false;
    selected: any;
    onStartControl: FormControl;
    onGroupsControl: FormControl;
    onAuthorControl: FormControl;

    // Notif cst
    notificationTypes = notificationTypes;
    notificationOnSuccess = notificationOnSuccess;
    notificationOnFailure = notificationOnFailure;

    // Only set to edit an existing notification
    isNewNotif = true;
    @Input('notification')
    set notification (data: Notification){
        if (data) {
            this.isNewNotif = false;
            this.selected.pipeline = [];
            this.selected.pipeline .push(data.pipeline.name);
            this.updateNeedEnv(this.selected.pipeline);
            this.selected.environment = [];
            this.selected.environment.push(data.environment.name);

            this.selected.type = Object.keys(data.notifications)[0];

            this.selected.notification = new UserNotificationSettings();
            this.selected.notification = data.notifications[this.selected.type];

            this.selected.recipients = this.selected.notification.recipients.join(',');

            this.onStartControl.setValue(this.selected.notification.on_start);
            this.onGroupsControl.setValue(this.selected.notification.send_to_groups);
            this.onAuthorControl.setValue(this.selected.notification.send_to_author);
        }
    }

    @ViewChild('myModal')
    private modal: SemanticModalComponent;
    @ViewChild('deleteButton')
    deleteButtonComponent: DeleteButtonComponent;

    constructor(private _translate: TranslateService) {
        this.initForm();
    }

    ngAfterViewInit(): void {
        this.modal.onModalShow.subscribe(() => {
            if (this.deleteButtonComponent) {
                this.deleteButtonComponent.reset();
            }
        });
    }

    // Init form
    initForm(): void {
        this.selected = {
            pipeline: [],
            environment: [],
            notification : new UserNotificationSettings(),
            type: notificationTypes[0],
            recipients: ''
        };
        this.onStartControl = new FormControl(this.selected.notification.on_start);
        this.onGroupsControl = new FormControl(this.selected.notification.send_to_groups);
        this.onAuthorControl = new FormControl(this.selected.notification.send_to_author);
    }

    // Check if need to select an environment
    updateNeedEnv(pip: Array<string>): void {
        this.needEnv = false;
        pip.forEach(pipName => {
            let pipeline = this.project.pipelines.find(p => p.name === pipName);
            if (pipeline && (pipeline.type === 'deployment' || pipeline.type === 'testingt'))  {
                this.needEnv = true;
            }
        });
    }

    // Show modal
    show(data?: {}) {
        this.modal.show(data);
    }

    removeNotif() {
        this.loading = true;
        let pipName = this.selected.pipeline[0];
        let envName = this.selected.environment[0];

        let currentNotif =  this.application.notifications.find(n => {
            return n.pipeline.name === pipName && n.environment.name === envName;

        });
        if (currentNotif) {
            delete currentNotif.notifications[this.selected.type];
            let notifs = new Array<Notification>();
            notifs.push(currentNotif);
            if (Object.keys(currentNotif.notifications).length === 0) {
                this.send(new NotificationEvent('delete', notifs));
            } else {
                this.send(new NotificationEvent('update', notifs));
            }
        }
    }

    sendEvent() {
        this.loading = true;
        let notifications = new Array<Notification>();
        this.selected.pipeline.forEach(pipName => {
            let appPipeline = this.application.pipelines.find(p => p.pipeline.name === pipName);
            if (appPipeline) {
                if (appPipeline.pipeline.type === 'deployment' || appPipeline.pipeline.type === 'testing') {
                    this.selected.environment.forEach(env => {
                        let completedEnv = this.project.environments.find(e => e.name === env);
                        if (completedEnv) {
                            notifications.push(this.createNotification(appPipeline, completedEnv));
                        }
                    });
                } else {
                    notifications.push(this.createNotification(appPipeline));
                }
            }
        });
        let type = 'update';
        if (this.isNewNotif) {
            type = 'add';
        }
        this.send(new NotificationEvent(type, notifications));
    }

    send(e: NotificationEvent): void {
        this.event.emit(e);
    }

    close() {
        this.initForm();
        this.modal.hide();
    }

    createNotification(appPip: ApplicationPipeline, environment?: Environment): Notification {
        let n: Notification;
        if (this.application.notifications) {
            n = this.application.notifications.find(notif => {
                if (environment) {
                    return notif.application_pipeline_id === appPip.id && environment.name === notif.environment.name;
                }
                return notif.application_pipeline_id === appPip.id;
            });
        }

        if (!n) {
            n = new Notification();
            n.application_pipeline_id = appPip.id;
            n.pipeline = appPip.pipeline;
            if (environment) {
                n.environment = environment;
            }
        }
        this.selected.notification.on_start = this.onStartControl.value;
        this.selected.notification.send_to_groups = this.onGroupsControl.value;
        this.selected.notification.send_to_author = this.onAuthorControl.value;
        this.selected.notification.recipients = this.selected.recipients.split(',');
        n.notifications[this.selected.type] = this.selected.notification;
        return n;
    }


    getTitle(): string {
        if (this.isNewNotif) {
            return this._translate.instant('application_notifications_form_title_add');
        }
        return this._translate.instant('application_notifications_form_title_edit');
    }
}
