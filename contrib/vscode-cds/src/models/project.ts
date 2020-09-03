export interface Project {
    key: string;
    name: string;
    description: string;
    icon: string;
    // workflows: Array<Workflow>;
    // workflow_names: Array<IdName>;
    // pipelines: Array<Pipeline>;
    // pipeline_names: Array<IdName>;
    // applications: Array<Application>;
    // application_names: Array<IdName>;
    // groups: Array<GroupPermission>;
    // variables: Array<Variable>;
    // environments: Array<Environment>;
    permission: number;
    last_modified: string;
    //vcs_servers: Array<RepositoriesManager>;
    //keys: Array<Key>;
    //integrations: Array<ProjectIntegration>;
    features: {};
    //labels: Label[];
    metadata: {};
    favorite: boolean;
    // true if someone has updated the project ( used for warnings )
    externalChange: boolean;
    loading: boolean;
    mute: boolean;
}
