import {Injectable} from '@angular/core';
import {BehaviorSubject} from 'rxjs/BehaviorSubject';
import {EventSubscription} from '../../model/event.model';
import {CDSWorker} from '../../shared/worker/worker';

@Injectable()
export class EventStore {

    worker: CDSWorker;
    _eventFilter: BehaviorSubject<EventSubscription> = new BehaviorSubject(null);

    constructor() {
    }

    init(worker: CDSWorker, uuid: string): void {
        this.worker = worker;
        let f = this._eventFilter.getValue();
        if (!f) {
            f = new EventSubscription();
            f.uuid = uuid;
        } else {
            f.uuid = uuid;
            this.changeFilter(f, false);

        }
        this._eventFilter.next(f);
    }

    changeFilter(filter: EventSubscription, sendToWorker: boolean) {
        if (this.worker) {
            filter.uuid = this._eventFilter.getValue().uuid;
        }
        // get previous filter
        let prev = this._eventFilter.getValue();
        if (prev) {
            // test proj/workflow
            if (prev.workflow_name === filter.workflow_name && prev.key === filter.key) {
                // keep event on last workflow run
                if (prev.num && !filter.num) {
                    return;
                }
            }
        }
        this._eventFilter.next(filter);
        if (this.worker && sendToWorker) {
            let msg = {
                add_filter: filter
            };
            this.worker.sendMsg(msg);
        }
    }
}
