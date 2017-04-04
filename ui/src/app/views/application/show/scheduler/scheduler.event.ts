import {Scheduler} from '../../../../model/scheduler.model';
export class SchedulerEvent {
    type: string;
    scheduler: Scheduler;

    constructor(t: string, s: Scheduler) {
        this.type = t;
        this.scheduler = s;
    }
}
