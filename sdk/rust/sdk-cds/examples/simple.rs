#[macro_use]
extern crate serde_derive;

use std::env;

use sdk_cds::client::Client;

fn main() {
    let cds_user = env::var("CDS_USER").expect("You must export environment variable named CDS_USER");
    let cds_host = env::var("CDS_HOST").expect("You must export environment variable named CDS_HOST");
    let cds_token = env::var("CDS_TOKEN").expect("You must export environment variable named CDS_TOKEN");
    let my_client = Client::new(cds_host.as_str(), cds_user.as_str(), cds_token.as_str());

    println!("Hello, world! {:?}", my_client.status().unwrap());
    println!("Me : {:?}", my_client.me().unwrap());
    println!("projects : {:?}", my_client.projects().unwrap());
    println!(
        "applications : {:?}",
        my_client.applications("TEST").unwrap()[0].name
    );
    println!(
        "application name : {:?}",
        my_client.application("TEST", "test").unwrap().icon
    );
    println!("workflows : {:?}", my_client.workflows("TEST").unwrap());
    println!(
        "workflow test : {:?}",
        my_client.workflow("TEST", "test").unwrap()
    );
}
