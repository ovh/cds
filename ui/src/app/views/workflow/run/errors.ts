export const ErrorMessageMap: { [key: string]: Message } = {
    'MsgWorkflowErrorBadPipelineName': {
        title: 'workflow_error_bad_pipeline_name_title',
        link: 'https://ovh.github.io/cds/docs/concepts/files/workflow-syntax/'
    },
    'MsgWorkflowErrorBadApplicationName': {
        title: 'workflow_error_bad_application_name_title',
        link: 'https://ovh.github.io/cds/docs/concepts/files/workflow-syntax/'
    },
    'MsgWorkflowErrorBadEnvironmentName': {
        title: 'workflow_error_bad_environment_name_title',
        link: 'https://ovh.github.io/cds/docs/concepts/files/workflow-syntax/'
    },
    'MsgWorkflowErrorBadIntegrationName': {
        title: 'workflow_error_bad_integration_name_title',
        link: 'https://ovh.github.io/cds/docs/concepts/files/workflow-syntax/'
    },
    'MsgWorkflowErrorBadCdsDir': {
        title: 'workflow_error_bad_cds_dir_title',
        description: 'workflow_error_bad_cds_dir_description',
        link: 'https://ovh.github.io/cds/docs/tutorials/init_workflow_with_cdsctl/'
    },
    'MsgWorkflowErrorUnknownKey': {
        title: 'workflow_error_unknown_key_title',
        description: 'workflow_error_unknown_key_description',
        link: 'https://ovh.github.io/cds/docs/tutorials/init_workflow_with_cdsctl/'
    },
    'MsgWorkflowErrorBadVCSStrategy': {
        title: 'workflow_error_bad_vcs_strategy_title',
        description: 'workflow_error_bad_vcs_strategy_description',
        link: 'https://ovh.github.io/cds/docs/concepts/files/application-syntax/'
    },
};

interface Message {
    title: string;
    description?: string;
    link: string
}
