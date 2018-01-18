export class VCSStrategy {
    connection_type: string;
    user: string;
    password: string;
    default_branch: string;
    branch: string;
    ssh_key: string;
    pgp_key: string;

    constructor() {
        this.connection_type = VCSConnections.HTTPS;
        this.default_branch = 'master';
        this.branch = '{{.git.branch}}';
    }
}

export class VCSConnections {
    static SSH = 'ssh';
    static HTTPS = 'https';

    static values(): Array<string> {
        return new Array<string>(VCSConnections.SSH, VCSConnections.HTTPS);
    }
}
