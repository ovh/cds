# CDS UI
 
CDS/UI is a webclient for CDS

This project was generated with [angular-cli](https://github.com/angular/angular-cli).

## Prerequisites

The project have dependencies that require Node.js 6.9.0 or higher, together with npm 3 or higher.

## Development server

Install the dependencies: `npm install`.

Run `npm start` to launch a development server. 

Navigate to `http://localhost:8080/`.

The app will automatically reload if you change any of the source files.

API URL can be change here: `src/environments/environment.ts`.

## Running unit tests

Run `npm test` to execute the unit tests.

## Running e2e tests

Export template files:

`export templateLogin=$(cat login.template)`

`export templateCreateProject=$(cat create.project.template)`

Run test:

`venom run loginUser.yml --var cds.build.url=<your_cds_ui_url> --var cds.build.user=<your_cds_user> --var cds.build.user_password=<your_cds_password> --output-dir results --details low`
