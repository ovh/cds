import { UiNg2Page } from './app.po';

describe('ui-ng2 App', function() {
  let page: UiNg2Page;

  beforeEach(() => {
    page = new UiNg2Page();
  });

  it('should display message saying app works', () => {
    page.navigateTo();
    expect(page.getParagraphText()).toEqual('app works!');
  });
});
