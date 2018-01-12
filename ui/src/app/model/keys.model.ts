export class Keys {
    project_key: Array<Key>;
    application_key: Array<Key>;
    environment_key: Array<Key>;

    static formatForSelect(keys: any): AllKeys {
        let k = new AllKeys();
        if (keys.project_key) {
            k.ssh.push(...keys.project_key.filter(key => key.type === KeyType.SSH).map(key => 'proj-' + key.name ));
            k.pgp.push(...keys.project_key.filter(key => key.type === KeyType.PGP).map(key => 'proj-' + key.name ));
        }
        if (keys.application_key) {
            k.ssh.push(...keys.application_key.filter(key => key.type === KeyType.SSH).map(key => 'app-' + key.name ));
            k.pgp.push(...keys.application_key.filter(key => key.type === KeyType.PGP).map(key => 'app-' + key.name ));
        }
        if (keys.environment_key) {
            k.ssh.push(...keys.environment_key.filter(key => key.type === KeyType.SSH).map(key => 'env-' + key.name ));
            k.pgp.push(...keys.environment_key.filter(key => key.type === KeyType.PGP).map(key => 'env-' + key.name ));
        }
        return k;
    }
}

export class AllKeys {
    ssh: Array<string>;
    pgp: Array<string>;

    constructor() {
        this.ssh = new Array<string>();
        this.pgp = new Array<string>();
    }
}

export class Key {
    name: string;
    public: string;
    private: string;
    key_id: string;
    type: string;
    application_id: number;
    pipeline_id: number;

    constructor() {
        this.name  = '';
    }
}

export class KeyType {
    static SSH = 'ssh';
    static PGP = 'pgp';

    static values(): Array<string> {
        let v = new Array<string>();
        v.push(KeyType.SSH);
        v.push(KeyType.PGP);
        return v;
    }
}
