
export interface Application {
    id: number;
    name: string;
    description: string;
    icon: string;
    // variables: Array<Variable>;
    // permission: number;
    // notifications: Array<Notification>;
    // last_modified: string;
    // vcs_server: string;
    // repository_fullname: string;
    // usage: Usage;
    // keys: Array<Key>;
    // vcs_strategy: VCSStrategy;
    // deployment_strategies: Map<string, any>;
    // vulnerabilities: Array<Vulnerability>;
    // project_key: string; // project unique key
    // from_repository: string;

    // // true if someone has updated the application ( used for warnings )
    // externalChange: boolean;

    // // workflow depth for horizontal tree view
    // horizontalDepth: number;
}
