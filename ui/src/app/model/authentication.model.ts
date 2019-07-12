import { Group } from './group.model';
import { User } from './user.model';

export class AuthDriverManifest {
    type: string;
    signup_disabled: boolean;
}

export class AuthConsumerSigninResponse {
    token: string;
    user: User;
}


export class AuthConsumer {
    id: string;
    name: string;
    description: string;
    parent_id: string;
    authentified_user_id: string;
    type: string;
    created: string;
    group_ids: Array<number>;
    scopes: Array<string>;
    groups: Array<Group>;

    // UI fields
    parent: AuthConsumer;
}

export class AuthSession {
    id: string;
    consumer_id: string;
    expire_at: string;
    created: string;
    group_ids: Array<number>;
    scopes: Array<string>;
    current: boolean;

    // UI fields
    consumer: AuthConsumer;
}

