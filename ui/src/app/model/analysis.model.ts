
export const StatusAnalyzeInProgress = "InProgress"
export const StatusAnalyzeSucceed = "Success"
export const StatusAnalyzeError = "Error"
export const StatusAnalyzeSkipped = "Skipped"

export class RepositoryAnalysis {
    id: string;
    created: Date;
    last_modified: Date;
    project_repository_id: string;
    vcs_project_id: string;
    project_key: string;
    status: string;
    branch: string;
    commit: string;
    data: AnalysisData;

}

export class AnalysisData {
    operation_uuid: string;
    commit_check: boolean;
    sign_key_id: string;
    cds_username: string;
    cds_username_id: string;
    error: string;
    entities: DataEntity[];
}

export class DataEntity {
    file_name: string;
    path: string;
    status: string;
}

export class AnalysisRequest  {
	projectKey: string;
	vcsName: string;
	repoName: string;
	ref: string;
}

export class AnalysisResponse {
    analysis_id: string;
    status: string;
}

export class Analysis {
    id: string;
    created: string;
    last_modified: string;
    project_repository_id: string;
    vcs_project_id: string;
    project_key: string;
    status: string;
    ref: string;
    commit: string;
    data: AnalysisData;
}