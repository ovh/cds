import config from '../config';
import { Selector, ClientFunction, t } from "testcafe";

export default class WorkflowCreatePage {

    url: string;
    nextButton: Selector;
    workflowNameInput: Selector;
    pipelineNameInput: Selector;
    appicationNameInput: Selector;

    steps: Selector;

    getLocation = ClientFunction(() => document.location.href);

    constructor(key: string) {
        this.url = config.baseUrl + '/project/' + key + '/workflow';
        this.nextButton = Selector('.ui.green.button');
        this.workflowNameInput = Selector('input[name=name]');
        this.pipelineNameInput = Selector('input[name=pipname]');
        this.appicationNameInput = Selector('input[name=appname]');
        this.steps = Selector('app-workflow-node-add-wizard .step.pointing')
    }

    async createWorkflow(workflowName: string, key: string, pipName: string, appName: string) {
        await t
            .expect(this.nextButton.hasAttribute('disabled')).ok()
            .typeText(this.workflowNameInput, workflowName)
            .click(this.nextButton)

            // Create pipeline
            .expect(this.nextButton.hasAttribute('disabled')).ok()
            .expect(this.steps.nth(0).classNames).contains('active')
            .typeText(this.pipelineNameInput, pipName)
            .click(this.nextButton)

            // Create application
            .expect(this.steps.nth(1).classNames).contains('active')
            .typeText(this.appicationNameInput, appName)
            .click(this.nextButton)

            // Create workflow
            .click(this.nextButton)
            .expect(this.getLocation()) .eql(this.url + '/' + workflowName);
    }
}
