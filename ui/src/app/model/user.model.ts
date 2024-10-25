import { AuthConsumer, AuthDriverManifest, AuthSession } from './authentication.model';

export class AuthSummary {
    user: AuthentifiedUser;
    consumer: AuthConsumer;
    session: AuthSession;
    driverManifest: AuthDriverManifest;

    constructor() { }

    isAdmin(): boolean {
        const dontNeedMFA = !this.driverManifest.support_mfa;
        return this.user.ring === 'ADMIN' && (dontNeedMFA || this.session.mfa);
    }

    isMaintainer(): boolean {
        return this.user.ring === 'MAINTAINER' || this.user.ring === 'ADMIN';
    }
}

export class AuthentifiedUser {
    id: string;
    created: string;
    username: string;
    fullname: string;
    ring: string;
    organization: string;
}

export class UserLink {
    username: string;
    type: string;
}

export class User {
    id: number;
    username: string;
    fullname: string;
    email: string;
    admin: boolean;
    token: string;
    password: string;
}

export class UserLoginRequest {
    username: string;
    password: string;
    request_token: string;
}

export class UserContact {
    id: number;
    created: string;
    user_id: string;
    type: string;
    value: string;
    primary: boolean;
    verified: boolean;
}

export class Schema {
    application: string;
    pipeline: string;
    environment: string;
    workflow: string;
}

export class UserGPGKey {
    id: string;
    authentified_user_id: string;
    key_id: string;
    public_key: string;
    created: string;
}