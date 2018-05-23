export class PlatformModel {
    id: number;
    name: string;
    author: string;
    identifier: string;
    icon: string;
    default_config: {};
    deployment_default_config: {};
    disabled: boolean;
    hook: boolean;
    file_storage: boolean;
    block_storage: boolean;
    deployment: boolean;
    compute: boolean;
    public: boolean;
}

export class ProjectPlatform {
    id: number;
    name: string;
    project_id: number;
    platform_model_id: number;
    model: PlatformModel;
    config: {};

    // UI attributes
    hasChanged = false;

    static mergeConfig(default_config: {}, config: {}) {
        if (!default_config) {
            return;
        }
        if (!config) {
            config = {};
        }
        for (let k of Object.keys(config)) {
            if (default_config[k] ==  null) {
                delete config[k];
            }
        }

        for (let k of Object.keys(default_config)) {
            if (config[k] == null) {
                config[k] = default_config[k];
            }
        }
    }

    constructor() {
        this.config = {};
    }
}
