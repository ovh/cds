export class Keys {
    project_key: Array<Key>;
    application_key: Array<Key>;
    environment_key: Array<Key>;

    static formatForSelect(keys: any): Array<string> {
        let k = new Array<string>();
        if (keys.project_key) {
            k.push(...keys.project_key.map(key => {
                return 'proj-' + key.name;
            }));
        }
        if (keys.application_key) {
            k.push(...keys.application_key.map(key => {
                return 'app-' + key.name;
            }));
        }
        if (keys.environment_key) {
            k.push(...keys.environment_key.map(key => {
                return 'env-' + key.name;
            }));
        }
        return k;
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
