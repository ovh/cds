import { ChangeDetectionStrategy, Component, EventEmitter, Input, Output } from '@angular/core';
import { ExpirationToString, Token, TokenEvent } from 'app/model/token.model';
import { Table } from 'app/shared/table/table';

@Component({
    selector: 'app-token-list',
    templateUrl: './token.list.html',
    styleUrls: ['./token.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class TokenListComponent extends Table<Token> {

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
    expirationToString = ExpirationToString;
    newToken: Token;

    constructor() {
        super();
        this.newToken = new Token();
    }

    getData(): Array<Token> {
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
