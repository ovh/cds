import {Token} from '../../model/token.model';

export class TokenEvent {
    type: string;
    token: Token;

    constructor(type: string, t: Token) {
        this.type = type;
        this.token = t;
    }
}
