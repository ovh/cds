import { ClientFunction, Selector, t } from 'testcafe';
import config from '../config';

export default class NavbarPage {

    createProjectLink: Selector;

    getLocation = ClientFunction(() => document.location.href);

    constructor() {
        this.createProjectLink = Selector('[href="/project"]');
    }

    async clickCreateProject() {
        await t
            .click(this.createProjectLink)
            .expect(this.getLocation()).eql(config.baseUrl + '/project');
    }
}
