import {Hook} from '../../../../model/hook.model';

export class HookEvent {
    type: string;
    hook: Hook;

    constructor(t: string, h: Hook) {
        this.type = t;
        this.hook = h;
    }
}
