export class Token {
  id: number;
  token: string;
  expiration: ExpirationTokenType;
  created: string;
  description: string;
  creator: string;
  group_name: string;

  // useful for ui
  updating: boolean;
  expirationString: string;
}

export enum ExpirationTokenType {
  session = 1,
  daily,
  persistent,
}

export const ExpirationToString = ['', 'session', 'daily', 'persistent'];

export class TokenEvent {
    type: string;
    token: Token;

    constructor(type: string, t: Token) {
        this.type = type;
        this.token = t;
    }
}
