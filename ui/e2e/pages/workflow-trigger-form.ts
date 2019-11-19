import { Selector, ClientFunction, t } from "testcafe";

export default class WorkflowTriggerFormPage {

    // Next button
    nextButton: Selector;

    // Select intput search
    inputSearch: Selector;

    // Create pipeline input text
    pipelineNameInput: Selector;

    // Create application input text
    appicationNameInput: Selector;

    // Steps menu
    steps: Selector;

    getLocation = ClientFunction(() => document.location.href);

    constructor() {
        this.nextButton = Selector('button.ui.green');
        this.steps = Selector('app-workflow-node-add-wizard .step.pointing');
        this.inputSearch = Selector('input.search');
        this.pipelineNameInput = Selector('input[name=pipname]');
        this.appicationNameInput = Selector('input[name=appname]');
    }

    async addExistingPipeline(pipName: string) {
        await t
            .expect(this.steps.nth(0).classNames).contains('active') // pipeline is active
            .click(Selector('div.ui.search.dropdown'))
            .click(Selector('div.ui.search.dropdown > div.menu > div.item').nth(0))
            .click(this.nextButton) // next step go to application
            .expect(this.steps.nth(1).classNames).contains('active') // application is active
            .click(this.nextButton) // next step go to env
            .expect(this.steps.nth(2).classNames).contains('active') // env is active
            .click(this.nextButton) // next step go to finish
    }
}
