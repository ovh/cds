import {Component, Input, ViewChild, Output, EventEmitter, AfterViewInit} from '@angular/core';
import {
    UserNotificationSettings,
    notificationTypes,
    notificationOnSuccess,
    notificationOnFailure,
    Notification
} from '../../../../../model/notification.model';
import {Project} from '../../../../../model/project.model';
import {Application, ApplicationPipeline} from '../../../../../model/application.model';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {FormControl} from '@angular/forms';
import {Environment} from '../../../../../model/environment.model';
import {NotificationEvent} from '../notification.event';
import {TranslateService} from '@ngx-translate/core';
import {DeleteButtonComponent} from '../../../../../shared/button/delete/delete.button';
import {AutoUnsubscribe} from '../../../../../shared/decorator/autoUnsubscribe';
import {ProjectService} from '../../../../../service/project/project.service';
import {ProjectStore} from '../../../../../service/project/project.store';
import {cloneDeep} from 'lodash';
import {Subscription} from 'rxjs/Subscription';
import {finalize} from 'rxjs/operators';

@Component({
    selector: 'app-notification-form-modal',
    templateUrl: './notification.form.html',
    styleUrls: ['./notification.form.scss']
})
@AutoUnsubscribe()
export class ApplicationNotificationFormModalComponent implements AfterViewInit {

    // Component output
    @Output() event = new EventEmitter<NotificationEvent>();

    // Component input
    @Input() project: Project;
    @Input() application: Application;
    @Input() loading = false;

    // existing notification
    projectNotifications: Array<Notification> = new Array<Notification>();
    clonedNotification: Notification;

    // Form data
    needEnv = false;
    selected: any;
    onStartControl: FormControl;
    onGroupsControl: FormControl;
    onAuthorControl: FormControl;

    // Notif cst
    notificationTypes: Array<string> = notificationTypes;
    notificationOnSuccess = notificationOnSuccess;
    notificationOnFailure = notificationOnFailure;

    // Only set to edit an existing notification
    isNewNotif = true;
    applicationsSubscribtion: Subscription;
    loadingApps = true;

    @Input('notification')
    set notification(data: Notification) {
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
    modal: SemanticModalComponent;
    @ViewChild('deleteButton')
    deleteButtonComponent: DeleteButtonComponent;

    constructor(private _translate: TranslateService, private _projectService: ProjectService, private _projectStore: ProjectStore) {
        this.initForm();
        if (this.notificationTypes.indexOf('clone') === -1) {
            this.notificationTypes.push('clone');
        }
    }

    ngAfterViewInit(): void {
        this.modal.onModalShow.subscribe(() => {
            this.applicationsSubscribtion = this._projectStore.getProjectApplicationsResolver(this.project.key)
                .pipe(
                    finalize(() => this.loadingApps = false)
                )
                .subscribe((proj) => this.project = proj);
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
            notification: new UserNotificationSettings(),
            type: notificationTypes[0],
            clonedType: '',
            recipients: ''
        };
        this.isNewNotif = true;
        this.onStartControl = new FormControl(this.selected.notification.on_start);
        this.onGroupsControl = new FormControl(this.selected.notification.send_to_groups);
        this.onAuthorControl = new FormControl(this.selected.notification.send_to_author);
    }

    // Check if need to select an environment
    updateNeedEnv(pip: Array<string>): void {
        this.needEnv = false;
        pip.forEach(pipName => {
            let appPip = this.application.pipelines.find(p => p.pipeline.name === pipName);
            if (appPip && (appPip.pipeline.type === 'deployment' || appPip.pipeline.type === 'testing')) {
                this.needEnv = true;
            }
        });
    }

    updateWithClonedNotification(index: number): void {
        this.isNewNotif = true;
        this.selected.clonedType = Object.keys(this.projectNotifications[index].notifications)[0];
        this.selected.notification = new UserNotificationSettings();
        this.selected.notification = this.projectNotifications[index].notifications[this.selected.clonedType];
        this.selected.recipients = this.selected.notification.recipients.join(',');
        this.onStartControl.setValue(this.selected.notification.on_start);
        this.onGroupsControl.setValue(this.selected.notification.send_to_groups);
        this.onAuthorControl.setValue(this.selected.notification.send_to_author);
    }

    getCloneNotificationLabel(n: Notification): string {
        let name = '';
        if (this.project) {
            name += this.project.applications.find(a => {
                if (a.pipelines.find(appPip => {
                        return appPip.id === n.application_pipeline_id;
                    })) {
                    return true;
                }
                return false;
            }).name;
            name += '-' + n.pipeline.name;
            if (n.environment && n.environment.name !== 'NoEnv') {
                name += '-' + n.environment.name;
            }
        }
        return '[' + Object.keys(n.notifications)[0] + '] ' + name;

    }

    // Show modal
    show(data?: {}) {
        this.modal.show(data);
        this._projectService.getAllNotifications(this.project.key).subscribe(ns => {
            if (ns && ns.length > 0) {
                this.projectNotifications = new Array<Notification>();
                ns.forEach(n => {
                    if (n.notifications) {
                        for (let key in n.notifications) {
                            if (n.notifications[key]) {
                                let notification = new Notification();
                                notification.application_pipeline_id = n.application_pipeline_id;
                                notification.pipeline = n.pipeline;
                                notification.environment = n.environment;
                                notification.notifications = {};
                                notification.notifications[key] = cloneDeep(n.notifications[key]);
                                this.projectNotifications.push(notification);
                            }
                        }
                    }
                });
            }
        });
    }

    removeNotif() {
        this.loading = true;
        let pipName = this.selected.pipeline[0];
        let envName = this.selected.environment[0];

        let currentNotif = this.application.notifications.find(n => {
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
        if (this.selected.type !== 'clone') {
            n.notifications[this.selected.type] = this.selected.notification;
        } else {
            n.notifications[this.selected.clonedType] = this.selected.notification;
        }

        return n;
    }


    getTitle(): string {
        if (this.isNewNotif) {
            return this._translate.instant('application_notifications_form_title_add');
        }
        return this._translate.instant('application_notifications_form_title_edit');
    }
}
