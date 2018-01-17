export class Token {
  id: number;
  token: string;
  expiration: ExpirationTokenType;
  created: string;
  description: string;
  creator: string;

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
