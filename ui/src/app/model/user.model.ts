export class AuthentifiedUser {
    id: string;
    created: string;
    username: string;
    fullname: string;
    ring: string;
    old_user_struct: User;

    constructor() { }

    isAdmin(): boolean {
        return this.ring === 'ADMIN';
    }
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
