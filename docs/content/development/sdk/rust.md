---
title: "SDK Rust"
card: 
  name: rest-sdk
---

## How to use it?

You have to initialize a cdsclient:

```rust
use std::env;

use sdk_cds::Client;

fn main() {
    let cds_host = "http://localhost:8081";
    let cds_token = "mytoken";

    let client = Client::new(cds_host, cds_token);
}
```

and then, you can use it:

```rust
// list projects
let projects = client.projects().unwrap();

// list applications of project with key TEST
let applications = client.applications("TEST").unwrap();

// list workflows
let workflows = client.workflows("TEST", "test").unwrap();
```

Go on https://docs.rs/sdk-cds/latest/sdk_cds/ to see all available funcs and documentations.
	

## Example

+ Create a file `main.rs` with this content:

```rust
use std::env;

use sdk_cds::Client;

fn main() {
    let cds_host =
        env::var("CDS_HOST").expect("You must export environment variable named CDS_HOST");
    let cds_token =
        env::var("CDS_TOKEN").expect("You must export environment variable named CDS_TOKEN");
    let my_client = Client::new(cds_host.as_str(), cds_token.as_str());

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


```

+ Build & run it: 

```bash
$ export CDS_HOST=http://localhost:8081
$ export CDS_TOKEN=mytoken
$ cargo run
```
