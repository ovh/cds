/// <reference types="Cypress" />

context('Workflows', () => {
    beforeEach(() => {
        // Login
        cy.visit('http://127.0.0.1:4200/account/login')
        cy.get('input[name=username]').type('')
        cy.get('input[name=password]').type('')
        cy.get('.ui.green.button').click()
        cy.location('pathname').should('include', 'home')
    })

    it('Create Project', () => {
        let projKey = 'CDSE2';
        let projectName = 'CDSE2E';
        let workflowName= 'simpleWorkflow';
        createProject(projectName, projKey);
        createWorkflow(projKey, workflowName, 'myfirstPipeline', 'myfirstApplication');
    })
})

// CREATE PROJECT
function createProject(name, key) {
    cy.get('[href="/project"]').click()
    cy.location('pathname').should('include', '/project')
    cy.get('input[name=projectname]').type(name)
    cy.get('input[name=projectkey]').should('have.value', key)
    cy.get('input.search').type('aa').type('{enter}')
    cy.get('button[name=createproject]').click()
    cy.location('pathname').should('include', '/project/' + key);
}

// CREATE WORKFLOW
function createWorkflow(key, name, pipName, appName) {
    cy.get('[href="/project/'+key+'/workflow"]').click()
    cy.location('pathname').should('include', '/project/'+key+'/workflow')
    cy.get('.ui.green.button').should('be.disabled');
    cy.get('input[name=name]').type(name);
    cy.get('.ui.green.button').click();

    // Create pipeline
    cy.get('.step.pointing').eq(1).should('have.class', 'active')
    cy.get('.ui.green.button').should('be.disabled');
    cy.get('app-workflow-node-add-wizard .step.pointing').first().should('have.class', 'active')
    cy.get('input[name=pipname]').type(pipName)
    cy.get('.ui.green.button').click()

    // Create application
    cy.get('app-workflow-node-add-wizard .step.pointing').eq(1).should('have.class', 'active')
    cy.get('input[name=appname]').type(appName)
    cy.get('.ui.green.button').click()

    // Create workflow
    cy.get('.ui.green.button').click()
    cy.location('pathname').should('include', '/project/CDSE2/workflow/simpleWorkflow')

}
