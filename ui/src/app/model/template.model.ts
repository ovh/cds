import {Parameter} from './parameter.model';
export class Template {
    id: number;
    name: string;
    description: string;
    params: Array<Parameter>;
    hook: boolean;
}

export class ApplyTemplateRequest {
    name: string;
    template: string;
    template_params: Array<Parameter>;
}
