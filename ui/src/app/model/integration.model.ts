export class IntegrationModel {
    id: number;
    name: string;
    author: string;
    identifier: string;
    icon: string;
    default_config: {};
    deployment_default_config: {};
    disabled: boolean;
    hook: boolean;
    storage: boolean;
    deployment: boolean;
    compute: boolean;
    event: boolean;
    public: boolean;
}

export class ProjectIntegration {
    id: number;
    name: string;
    project_id: number;
    integration_model_id: number;
    model: IntegrationModel;
    config: {};

    // UI attributes
    hasChanged = false;

    constructor() {
        this.config = {};
    }

    static mergeConfig(default_config: {}, config: {}) {
        if (!default_config) {
            return;
        }
        if (!config) {
            config = {};
        }
        for (let k of Object.keys(config)) {
            if (default_config[k] == null) {
                delete config[k];
            }
        }

        for (let k of Object.keys(default_config)) {
            if (config[k] == null) {
                config[k] = default_config[k];
            }
        }
    }
}
