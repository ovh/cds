import {Component, Input, EventEmitter, Output} from '@angular/core';
import {Token, ExpirationToString} from '../../../model/token.model';
import {Table} from '../../table/table';
import {TokenEvent} from '../token.event.model';

@Component({
    selector: 'app-token-list',
    templateUrl: './token.list.html',
    styleUrls: ['./token.list.scss']
})
export class TokenListComponent extends Table {

    @Input('tokens')
    set tokens(data: Token[]) {
        this._tokens = data;
        this.goTopage(1);
    }
    get tokens() {
      return this._tokens;
    }
    @Input('maxPerPage')
    set maxPerPage(data: number) {
        this.nbElementsByPage = data;
    };
    @Input() displayAdd: boolean;

    @Output() event = new EventEmitter<TokenEvent>();

    public ready = false;
    private _tokens: Token[];
    filter: string;
    showToken: {} = {};
    expirationToString = ExpirationToString;
    newToken: Token;

    constructor() {
        super();
        this.newToken = new Token();
    }

    getData(): any[] {
        if (!this.filter || this.filter === '') {
            return this.tokens;
        } else {
            return this.tokens.filter(v => v.description.toLowerCase().indexOf(this.filter.toLowerCase()) !== -1);
        }
    }

    /**
     * Send Event to parent component.
     * @param type Type of event (update, delete)
     * @param variable Variable data
     */
    sendEvent(type: string, token: Token): void {
        token.updating = true;
        this.event.emit(new TokenEvent(type, token));
    }
}
