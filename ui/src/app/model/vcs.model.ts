export class VCSStrategy {
    connection_type: string;
    user: string;
    password: string;
    ssh_key: string;
    pgp_key: string;

    constructor() {
        this.connection_type = VCSConnections.HTTPS;
    }
}

export class VCSConnections {
    static SSH = 'ssh';
    static HTTPS = 'https';

    static values(): Array<string> {
        return new Array<string>(VCSConnections.SSH, VCSConnections.HTTPS);
    }
}
