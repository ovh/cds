#[macro_use]
extern crate serde_derive;

use sdk_cds::client::Client;

fn main() {
    let my_client = Client::new("http://localhost:8081", "admin", "XXX-XXX-XXX-XXX-XXX");

    println!("Hello, world! {:?}", my_client.status().unwrap());
    println!("Me : {:?}", my_client.me().unwrap());
    println!("projects : {:?}", my_client.projects().unwrap());
    println!(
        "applications : {:?}",
        my_client.applications("TEST").unwrap()[0].name
    );
    println!(
        "application name : {:?}",
        my_client.application("TEST", "mytest").unwrap().icon
    );
    println!("workflows : {:?}", my_client.workflows("TEST").unwrap());
    println!(
        "workflow mytest : {:?}",
        my_client.workflow("TEST", "mytest").unwrap()
    );
}
