export class User {
    id: number;
    username: string;
    fullname: string;
    email: string;
    admin: boolean;
    token: string;
    password: string;

    constructor() {}
}

export class Token {
  token: string;
  expiration: ExpirationTokenType;
  created: string;
  description: string;
  creator: string;

  // useful for ui
  deleting: boolean;
}

export enum ExpirationTokenType {
  Session = 1,
  Daily,
  Persistent,
}
