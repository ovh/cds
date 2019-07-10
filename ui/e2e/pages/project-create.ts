import config from '../config';
import { Selector, ClientFunction, t } from "testcafe";

export default class ProjectCreatePage {

  url: string;
  projectNameInput: Selector;
  projectKeyInput: Selector;
  groupSelect: Selector;
  createButton: Selector;

  getLocation = ClientFunction(() => document.location.href);

  constructor() {
    this.url = config.baseUrl + '/project';
    this.projectNameInput = Selector('input[name=projectname]');
    this.projectKeyInput = Selector('input[name=projectkey]');
    this.groupSelect = Selector('input.search');
    this.createButton = Selector('button[name=createproject]');
  }

  async createProject(name: string, key: string, group: string) {
    await t
      .typeText(this.projectNameInput, name)
      .expect(this.projectKeyInput.value).eql(key)
      .typeText(this.groupSelect, group).pressKey('enter')
      .click(this.createButton)
      .expect(this.getLocation()).eql(config.baseUrl + '/project/' + key);
  }
}
