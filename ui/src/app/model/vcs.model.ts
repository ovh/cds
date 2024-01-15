export class VCSProject {
    id: string;
    name: string;
    auth: VCSProjectAuth;
    options: VCSProjectOptions;
    type: string;
	created: string;
	lastModified: string;
	createdBy: string;
	description: string;
	url: string;
}

export class VCSProjectOptions {
	disableWebhooks: boolean;
	disableStatus: boolean;
	disableStatusDetails: boolean;
	disablePolling: boolean;
	urlAPI: string;
}

export class VCSProjectAuth {
    username: string;
    token: string;
    sshKeyName: string;

    // Use for gerrit
    sshUsername:   string;
    sshPort:       number;
    sshPrivateKey: string;
}

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
