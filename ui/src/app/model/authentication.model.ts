import { WithKey } from 'app/shared/table/data-table.component';
import { Group } from './group.model';
import { AuthentifiedUser } from './user.model';

export class AuthConsumerScopeDetail {
    scope: string;
    endpoints: Array<AuthConsumerScopeEndpoint>;
}

export class AuthConsumerScopeEndpoint {
    route: string;
    methods: Array<string>;
}

export class AuthScope implements WithKey {
    value: string;

    constructor(value: string) {
        this.value = value;
    }

    key(): string {
        return this.value;
    }
}

export class AuthDriverManifests {
    is_first_connection: boolean;
    manifests: Array<AuthDriverManifest>;
}

export class AuthDriverSigningRedirect {
    method: string;
    url: string;
    body: any;
    content_type: string;
}

export class AuthDriverManifest {
    type: string;
    signup_disabled: boolean;
    support_mfa: boolean;

    // ui fields
    icon: string;
}

export class AuthCurrentConsumerResponse {
    user: AuthentifiedUser;
    consumer: AuthConsumer;
    session: AuthSession;
    driver_manifest: AuthDriverManifest;
}

export class AuthConsumerSigninResponse {
    token: string;
    user: AuthentifiedUser;
}

export class AuthConsumerCreateResponse {
    token: string;
    consumer: AuthConsumer;
}

export class AuthConsumerWarning {
    type: string;
    group_id: number;
    group_name: string;
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
    scope_details: Array<AuthConsumerScopeDetail>;
    groups: Array<Group>;
    disabled: boolean;
    warnings: Array<AuthConsumerWarning>;

    // UI fields
    parent: AuthConsumer;
    children: Array<AuthConsumer>;
    sessions: Array<AuthSession>;
}

export class AuthSession {
    id: string;
    consumer_id: string;
    expire_at: string;
    created: string;
    mfa: boolean;
    current: boolean;
    last_activity: string;

    // UI fields
    consumer: AuthConsumer;
}

