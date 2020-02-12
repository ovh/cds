use std::env;

use async_std::task;
use sdk_cds::Client;

fn main() {
    let cds_host =
        env::var("CDS_HOST").expect("You must export environment variable named CDS_HOST");
    let cds_token =
        env::var("CDS_TOKEN").expect("You must export environment variable named CDS_TOKEN");
    let my_client = Client::new(cds_host.as_str(), cds_token.as_str());

    println!(
        "Hello, world! {:?}",
        task::block_on(my_client.status()).unwrap()
    );
    println!("Me : {:?}", task::block_on(my_client.me()).unwrap());
    let projects = task::block_on(my_client.projects()).unwrap();
    println!("projects length : {}", projects.len());

    if projects.is_empty() {
        println!("no projects available");
        return;
    }
    let project_key = &projects.get(0).unwrap().key;

    let applications = task::block_on(my_client.applications(project_key)).unwrap();
    println!("applications length {}", applications.len());

    if applications.is_empty() {
        println!("no applications in project");
    } else {
        let application = task::block_on(my_client.application(
            &projects.get(0).unwrap().key,
            &applications.get(0).unwrap().name,
        ))
        .unwrap();
        println!("application name : {:?}", application.name);
    }

    let workflows = task::block_on(my_client.workflows(project_key)).unwrap();
    println!("workflows length : {}", workflows.len());

    if workflows.is_empty() {
        println!("no workflows found in project");
        return;
    }

    let workflow =
        task::block_on(my_client.workflow(project_key, &workflows.get(0).unwrap().name)).unwrap();
    println!("workflow name : {:?}", workflow.name);
}
