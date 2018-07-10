export class NavbarData {
    projects: Array<NavbarProjectData>;
}

export class NavbarProjectData {
    key: string;
    name: string;
    description: string;
    application_name: string;
    workflow_name: string;
    type: string;
    favorite: boolean;
}

export class NavbarRecentData {
    project_key: string;
    name: string; // workflow name
}

export class NavbarSearchItem {
  type: string;
  value: string;
  title: string;
  projectKey: string;
  favorite?: boolean;
}
