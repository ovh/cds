import config from '../config';
import { Selector, ClientFunction, t } from "testcafe";

export default class ProjectPage {

    url: string;
    createWorkflowLink: Selector;

    getLocation = ClientFunction(() => document.location.href);

    constructor(key: string) {
        this.url = config.baseUrl + '/project/' + key;
        this.createWorkflowLink = Selector('[href="/project/'+key+'/workflow"]');
    }

    async clickCreateWorkflow() {
        await t
            .click(this.createWorkflowLink)
            .expect(this.getLocation()).eql(this.url + '/workflow');
    }
}
