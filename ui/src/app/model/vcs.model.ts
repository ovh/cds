export class VCSStrategy {
    repoType: string;
    connectionType: string;
    user: string;
    password: string;
    defaultBranch: string;
    branch: string;
    sshKey: string;
    pgpKey: string;
    url: string;
    defaultDirectory: string;

    constructor() {
        this.repoType = 'git';
        this.connectionType = VCSConnections.HTTPS;
        this.defaultBranch = 'master';
        this.branch = '{{.git.branch}}';
        this.url = '{{git.http_url}}';
        this.defaultDirectory = '{{.cds.workspace}}';
    }
}

export class VCSConnections {
    static SSH = 'ssh';
    static HTTPS = 'https';

    static values(): Array<string> {
        return new Array<string>(VCSConnections.SSH, VCSConnections.HTTPS);
    }
}
