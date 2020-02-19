export function formatKeysForSelect(...keys: Key[]): AllKeys {
    let k = new AllKeys();
    k.ssh.push(...keys.filter(key => key.type === KeyType.SSH));
    k.pgp.push(...keys.filter(key => key.type === KeyType.PGP));
    return k;
}


export class AllKeys {
    ssh: Array<Key>;
    pgp: Array<Key>;

    constructor() {
        this.ssh = new Array<Key>();
        this.pgp = new Array<Key>();
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
        this.name = '';
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
