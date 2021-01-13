import { AuthConsumer, AuthSession } from './authentication.model';

export class AuthSummary {
    user: AuthentifiedUser;
    consumer: AuthConsumer;
    session: AuthSession;

    constructor() { }

    isComplete(): boolean {
        return !!this.user && !!this.consumer && !!this.session;
    }

    isAdmin(): boolean {
        if (!this.isComplete()) {
            return false;
        }
        const dontNeedMFA = !this.consumer.support_mfa;
        return this.user.ring === 'ADMIN' && (dontNeedMFA || this.session.mfa);
    }

    isMaintainer(): boolean {
        return this.isComplete()
            && (this.user.ring === 'MAINTAINER' || this.user.ring === 'ADMIN');
    }

    isMFAavailable(): boolean {
        return this.isComplete() && this.consumer.support_mfa && !this.session.mfa;
    }
}

export class AuthentifiedUser {
    id: string;
    created: string;
    username: string;
    fullname: string;
    ring: string;
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
