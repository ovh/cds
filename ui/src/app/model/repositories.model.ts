export class RepositoriesManager {
    id: number;
    type: string;
    name: string;
    url: string;
}

export interface Repository {
    id: number;
    name: string;
    slug: string;
    fullname: string;
    url: string;
    http_url: string;
    ssh_url: string;
};

export class Branch {
    id: number;
    display_id: string;
    latest_commit: string;
    default: boolean;

    constructor() {
        this.default = false;
        this.display_id = ' ';
    }
};

export interface Commit {
    id: string;
    author: Author;
    authorTimestamp: number;
    message: string;
    url: string;
};

export interface Author {
    name: string;
    displayName: string;
    email: string;
    avatar: string;
};
