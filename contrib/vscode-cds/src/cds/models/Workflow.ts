export interface CdsWorkflowRun {
  id: string;
  runNumber: number;
  status: string;
  started: string;
  workflowName: string;
  projectKey: string;
  username?: string;
}

export interface CDSWorkflow {
    name: string;
}
