import { User } from './user.model';

export class AuthConsumerSigninResponse {
  token: string;
  user: User;
}
