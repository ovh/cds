import { Help } from '../model/help.model';

export class AddHelp {
    static readonly type = '[Help] Add help';
    constructor(public payload: Help) { }
}
