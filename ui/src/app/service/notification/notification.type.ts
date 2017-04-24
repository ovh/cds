export type Permission = 'denied' | 'granted' | 'default';

export interface NotificationOpts {
    body?: string;
    icon?: string;
    tag?: string;
    data?: any;
    renotify?: boolean;
    silent?: boolean;
    sound?: string;
    noscreen?: boolean;
    sticky?: boolean;
    dir?: 'auto' | 'ltr' | 'rtl';
    lang?: string;
    vibrate?: number[];
    requireInteraction?: boolean;
    onclick?: any;
    onshow?: any;
    onerror?: any;
    onclose?: any;
}
