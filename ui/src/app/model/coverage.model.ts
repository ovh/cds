export class Coverage {
    workflow_id: number;
    workflow_node_run_id: number;
    workflow_run_id: number;
    run_number: number;
    repository: string;
    branch: string;
    report: CoverageReport;
    trend: CoverageTrend;
}

export class CoverageTrend {
    current_branch_report: CoverageReport;
    default_branch_report: CoverageReport;
}

export class CoverageReport {
    files: CoverageReportFile;
    total_lines: number;
    covered_lines: number;
    total_functions: number;
    covered_functions: number;
    total_branches: number;
    covered_branches: number;
}

export class CoverageReportFile {
    path: string;
    total_lines: number;
    covered_lines: number;
    total_functions: number;
    covered_functions: number;
    total_branches: number;
    covered_branches: number;
}
