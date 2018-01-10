export class Key {
    name: string;
    public: string;
    private: string;
    key_id: string;
    type: string;

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
