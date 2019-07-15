export class User {
    id: number;
    username: string;
    fullname: string;
    email: string;
    admin: boolean;
    token: string;
    password: string;

    constructor() { }
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
