export class NavbarData {
    projects: Array<NavbarProjectData>;
}

export class NavbarProjectData {
    key: string;
    name: string;
    application_names: Array<string>;
    workflow_names: Array<string>;
}

export class NavbarRecentData {
    project_key: string;
    name: string; // workflow name
}
