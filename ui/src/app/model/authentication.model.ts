import { User } from './user.model';

export class AuthDriverManifest {
  type: string;
  signup_disabled: boolean;
}

export class AuthConsumerSigninResponse {
  token: string;
  user: User;
}
