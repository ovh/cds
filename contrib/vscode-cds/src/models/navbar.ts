export interface NavbarData {
    projects: Array<NavbarProjectData>;
}

export interface NavbarProjectData {
    key: string;
    name: string;
    description: string;
    application_name: string;
    workflow_name: string;
    type: string;
    favorite: boolean;
}

export interface NavbarRecentData {
    project_key: string;
    name: string; // workflow name
}

export interface NavbarSearchItem {
    type: string;
    value: string;
    title: string;
    projectKey: string;
    favorite?: boolean;
}
