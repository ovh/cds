import {Token} from '../../model/user.model';

export class TokenEvent {
    type: string;
    token: Token;

    constructor(type: string, t: Token) {
        this.type = type;
        this.token = t;
    }
}
