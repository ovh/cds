import config from '../config';
import { Selector, ClientFunction, t } from "testcafe";

export default class WorkflowPage {

    url: string;
    workflowTab: Selector;
    runWorkflowButton: Selector;

    // Modal
    modalRunWorkflowButton: Selector;

    constructor(key: string, workflowName: string) {
        this.url = config.baseUrl + '/project/' + key + '/workflow/' + workflowName;
        this.workflowTab = Selector('#WorkflowGraphTabs');
        this.runWorkflowButton = Selector('.ui.green.buttons>.ui.button')
        this.modalRunWorkflowButton = Selector('sui-modal-dimmer .ui.green.button')
    }

    async runWorkflow() {
        await t
            .click(this.runWorkflowButton)
            .click(this.modalRunWorkflowButton);
    }

}
