import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {NotificationOpts, Permission} from './notification.type';

declare const Notification: any;

@Injectable()
export class NotificationService {

    permission: Permission;

    constructor() {
        this.permission  = this.isSupported() ? Notification.permission : 'denied';
    }

    requestPermission() {
        if ('Notification' in window) {
            Notification.requestPermission((status: any) => this.permission = status);
        }
    }

    isSupported() {
        return 'Notification' in window;
    }

    create(title: string, options?: NotificationOpts): any {
        options = Object.assign({}, {
            icon: 'assets/images/cds.png',
            requireInteraction: false
        }, options);

        return new Observable((obs: any) => {

            if (!('Notification' in window)) {
                obs.complete();
            }

            if (this.permission !== 'granted') {
                obs.complete();
            }

            const notif = new Notification(title, options);

            notif.onshow = (e: any) => {
                if (options.onshow && typeof options.onshow === 'function') {
                    options.onshow(e);
                }
                obs.next({notification: notif, event: e});
            };

            notif.onclick = (e: any) => {
                if (options.onclick && typeof options.onclick === 'function') {
                    options.onclick(e);
                } else {
                    window.focus();
                    notif.close();
                }
                obs.next({notification: notif, event: e});
            };

            notif.onerror = (e: any) => {
                if (options.onerror && typeof options.onerror === 'function') {
                    options.onerror(e);
                }
                obs.error({notification: notif, event: e});
            };

            notif.onclose = () => {
                if (options.onclose && typeof options.onclose === 'function') {
                    options.onclose();
                }
                obs.complete();
            };
        });
    }
}
