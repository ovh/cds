export class TimelineFilter {
    projects: { [key: string]: ProjectFilter; };
    allProjects = true;

    constructor() {
        this.projects = {};
    }
}

export class ProjectFilter {
    key: string;
    workflowName: { [key: string]: boolean; };
    allWorkflows = true;
    applicationName: { [key: string]: boolean; };
    allApplications = true;
    pipelineName: { [key: string]: boolean; };
    allPipelines = true;
    environmentName: { [key: string]: boolean; };
    allEnvironments = true;

    constructor() {
        this.workflowName = {};
        this.applicationName = {};
        this.pipelineName = {};
        this.environmentName = {};
    }
}
