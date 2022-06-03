use crate::auth;
use crate::error::{ApiError, Error as CdsError};
use crate::models;
use crate::service::Service;

use std::cell::RefCell;
use std::collections::HashMap;

use async_std::task;
use chrono::{DateTime, TimeZone, Utc};
use futures::prelude::*;
use parking_lot::RwLock;
use regex::Regex;
use serde::de::DeserializeOwned;
use serde::Serialize;
use surf;
use url::Url;

pub type Result<T> = std::result::Result<T, CdsError>;

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
/// Client to request CDS API
pub struct Client {
    pub name: String, //Useful for multi instance to give a name to your instance
    pub host: String,
    pub token: String,
    #[serde(skip)]
    session_token: RwLock<RefCell<String>>,
    pub hash: String,
    pub insecure_skip_verify_tls: bool,
}

impl Client {
    /// Create a new client to access CDS API
    pub fn new<T: Into<String>>(host: T, token: T) -> Self {
        let host: String = host.into();
        Client {
            host: host.clone(),
            token: token.into(),
            insecure_skip_verify_tls: !host.starts_with("https"),
            ..Default::default()
        }
    }

    /// Get CDS Status
    pub async fn status(&self) -> Result<models::MonitoringStatus> {
        let body: Vec<u8> = vec![];
        self.stream_json("GET".to_string(), "/mon/status".to_string(), body)
            .await
    }

    pub async fn config(&self) -> Result<HashMap<String, String>> {
        let body: Vec<u8> = vec![];
        self.stream_json("GET".to_string(), "/config/user".to_string(), body)
            .await
    }

    /// Get minimal information about current user
    pub async fn me(&self) -> Result<models::User> {
        let body: Vec<u8> = vec![];
        self.stream_json("GET".to_string(), String::from("/user/me"), body)
            .await
    }

    /// Get the list of projects
    pub async fn projects(&self) -> Result<Vec<models::Project>> {
        let body: Vec<u8> = vec![];
        self.stream_json("GET".to_string(), "/project".to_string(), body)
            .await
    }

    /// Get the list of applications in a project
    pub async fn applications(&self, project_key: &str) -> Result<Vec<models::Application>> {
        let body: Vec<u8> = vec![];
        self.stream_json(
            "GET".to_string(),
            format!("/project/{}/applications", project_key),
            body,
        )
        .await
    }

    /// Get the application's details given the project key and the application name
    pub async fn application(
        &self,
        project_key: &str,
        application_name: &str,
    ) -> Result<models::Application> {
        let body: Vec<u8> = vec![];
        self.stream_json(
            "GET".to_string(),
            format!("/project/{}/application/{}", project_key, application_name),
            body,
        )
        .await
    }

    /// Get all the workflows inside a project given his project key
    pub async fn workflows(&self, project_key: &str) -> Result<Vec<models::Workflow>> {
        let body: Vec<u8> = vec![];
        self.stream_json(
            "GET".to_string(),
            format!("/project/{}/workflows", project_key),
            body,
        )
        .await
    }

    /// Get the workflow's details given his name
    pub async fn workflow(
        &self,
        project_key: &str,
        workflow_name: &str,
    ) -> Result<models::Workflow> {
        let body: Vec<u8> = vec![];
        self.stream_json(
            "GET".to_string(),
            format!("/project/{}/workflows/{}", project_key, workflow_name),
            body,
        )
        .await
    }

    /// Fetch the count of the queue
    pub async fn queue_count(&self) -> Result<models::QueueCount> {
        let body: Vec<u8> = vec![];
        self.stream_json(
            "GET".to_string(),
            "/queue/workflows/count".to_string(),
            body,
        )
        .await
    }

    /// Get all personal bookmarks
    pub async fn bookmarks(&self) -> Result<Vec<models::Bookmark>> {
        let body: Vec<u8> = vec![];
        self.stream_json("GET".to_string(), String::from("/bookmarks"), body)
            .await
    }

    /// Get last workflow run given their workflow name
    pub async fn last_run(
        &self,
        project_key: &str,
        workflow_name: &str,
    ) -> Result<models::WorkflowRun> {
        let body: Vec<u8> = vec![];
        self.stream_json(
            "GET".to_string(),
            format!(
                "/project/{}/workflows/{}/runs/latest",
                project_key, workflow_name
            ),
            body,
        )
        .await
    }

    /// Useful to register a new service to the API
    pub async fn service_register(&self, service: &Service) -> Result<String> {
        self.stream_json::<_, Service>(
            "POST".to_string(),
            "/services/register".to_string(),
            service,
        )
        .await
        .map(|serv_resp: Service| serv_resp.hash)
    }

    async fn login(
        &self,
        consumer_type: String,
        body: HashMap<String, String>,
    ) -> Result<models::AuthConsumerSigninResponse> {
        self.stream_json(
            "POST".to_string(),
            format!("/auth/consumer/{}/signin", consumer_type),
            body,
        )
        .await
    }

    fn has_valid_token(&self) -> Result<bool> {
        let session_token = self.session_token.read();
        let session_token = &*session_token.borrow();
        if session_token.is_empty() {
            return Ok(false);
        }

        let token: jwt::TokenData<auth::AuthClaims> = jwt::dangerous_unsafe_decode(session_token)?;
        let expired_at: DateTime<Utc> = Utc.timestamp(token.claims.expires_at, 0);

        if expired_at < Utc::now() && token.claims.expires_at != 0 {
            Ok(false)
        } else {
            Ok(true)
        }
    }

    pub async fn stream_json<T: Serialize, U: DeserializeOwned>(
        &self,
        method: String,
        path: String,
        body: T,
    ) -> Result<U> {
        let uri = format!("{}{}", self.host, path);

        let url = Url::parse(uri.as_str()).expect("cannot parse url");
        let mut req_http = surf::Request::new(http::Method::from_bytes(method.as_bytes())?, url)
            .set_header("Content-Type", "application/json")
            .set_header("User-Agent", "CDS/sdk")
            .set_header("X-Requested-With", "X-CDS-SDK");

        let check_token = !uri.contains("/auth/consumer/builtin/signin")
            && !uri.contains("/auth/consumer/local/signin")
            && !uri.contains("/auth/consumer/local/signup")
            && !uri.contains("/auth/consumer/local/verify")
            && !uri.contains("/auth/consumer/worker/signin");

        if check_token && !self.has_valid_token()? && !self.token.is_empty() {
            // Renew session
            let mut body = HashMap::new();
            body.insert(String::from("token"), self.token.clone());

            let res = task::block_on(self.login(String::from("builtin"), body))?;
            let session_token = self.session_token.read();
            session_token.replace(res.token);
        }
        let rx_signin_routes = Regex::new(r#"/auth/consumer/.*/signin"#).unwrap();

        //No auth on signing routes
        if uri.starts_with(&self.host) && !rx_signin_routes.is_match(&uri) {
            let session_token = self.session_token.read();
            // auth the request
            req_http = req_http.set_header(
                "Authorization",
                format!("Bearer {}", &*session_token.borrow()),
            );
        }
        let mut resp_http = req_http.body_json(&body)?.await?;

        if resp_http.status().as_u16() > 400u16 {
            let mut err: ApiError = resp_http.body_json::<ApiError>().await?;
            err.status = resp_http.status().as_u16();
            return Err(CdsError::ApiError(err));
        }

        resp_http.body_json().map_err(CdsError::from).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use async_std::prelude::*;
    use async_std::task;
    use mockito::mock;

    #[test]
    fn test_new_token() {
        let auth_claims = auth::AuthClaims {
            id: String::from("3a95edaa-b78f-4d4d-b85d-9ea3f4d71740"),
            expires_at: 0,
            jti: String::from("3a95edaa-b78f-4d4d-b85d-9ea3f4d71740"),
            issued_at: 1580721850,
            issuer: String::from("cds-api"),
            subject: String::from("a0dfd977-b95c-4cbb-b62f-f30f9b65084c"),
            audience: None,
            not_before: None,
        };

        let token = jwt::encode(
            &jwt::Header::default(),
            &auth_claims,
            &jwt::EncodingKey::from_secret("secret".as_ref()),
        )
        .unwrap();

        let _m_signin = mock("POST", "/auth/consumer/builtin/signin")
            .with_status(200)
            .with_header("Content-Type", "application/json")
            .with_header("User-Agent", "CDS/sdk")
            .with_header("X-Requested-With", "X-CDS-SDK")
            .with_body(format!(
                r#"{{"api_url": "localhost", "token": "{}"}}"#,
                token
            ))
            .create();

        let _m_user_me = mock("GET", "/user/me")
            .with_status(200)
            .with_header("Content-Type", "application/json")
            .with_header("User-Agent", "CDS/sdk")
            .with_header("X-Requested-With", "X-CDS-SDK")
            .with_header("Authorization", &format!("Bearer {}", token))
            .with_body(r#"{"username": "me", "fullname": "me", "created": "2019-10-11T10:23:33.207715+02:00", "id": "xxx", "ring": "user"}"#)
            .create();

        let my_client = Client::new(&mockito::server_url(), &String::from("xxx"));

        let me = task::block_on(my_client.me()).unwrap();

        assert_eq!(me.username, String::from("me"));
        assert_eq!(me.fullname, String::from("me"));
        assert_eq!(me.ring, String::from("user"));
    }

    #[test]
    fn test_renew_token() {
        let auth_claims = auth::AuthClaims {
            id: String::from("3a95edaa-b78f-4d4d-b85d-9ea3f4d71740"),
            expires_at: 1000,
            jti: String::from("3a95edaa-b78f-4d4d-b85d-9ea3f4d71740"),
            issued_at: 1580721850,
            issuer: String::from("cds-api"),
            subject: String::from("a0dfd977-b95c-4cbb-b62f-f30f9b65084c"),
            audience: None,
            not_before: None,
        };

        let token = jwt::encode(
            &jwt::Header::default(),
            &auth_claims,
            &jwt::EncodingKey::from_secret("secret".as_ref()),
        )
        .unwrap();

        let _m_signin = mock("POST", "/auth/consumer/builtin/signin")
            .with_status(200)
            .with_header("Content-Type", "application/json")
            .with_header("User-Agent", "CDS/sdk")
            .with_header("X-Requested-With", "X-CDS-SDK")
            .with_body(format!(
                r#"{{"api_url": "localhost", "token": "{}"}}"#,
                token
            ))
            .create();

        let _m_user_me = mock("GET", "/user/me")
            .with_status(200)
            .with_header("Content-Type", "application/json")
            .with_header("User-Agent", "CDS/sdk")
            .with_header("X-Requested-With", "X-CDS-SDK")
            .with_header("Authorization", &format!("Bearer {}", token))
            .with_body(r#"{"username": "me", "fullname": "me", "created": "2019-10-11T10:23:33.207715+02:00", "id": "xxx", "ring": "user"}"#)
            .create();

        let my_client = Client::new(&mockito::server_url(), &String::from("xxx"));

        let me = task::block_on(my_client.me()).unwrap();

        assert_eq!(me.username, String::from("me"));
        assert_eq!(me.fullname, String::from("me"));
        assert_eq!(me.ring, String::from("user"));
    }

    #[test]
    fn test_good_token() {
        let auth_claims = auth::AuthClaims {
            id: String::from("3a95edaa-b78f-4d4d-b85d-9ea3f4d71740"),
            expires_at: 0,
            jti: String::from("3a95edaa-b78f-4d4d-b85d-9ea3f4d71740"),
            issued_at: 1580721850,
            issuer: String::from("cds-api"),
            subject: String::from("a0dfd977-b95c-4cbb-b62f-f30f9b65084c"),
            audience: None,
            not_before: None,
        };

        let token = jwt::encode(
            &jwt::Header::default(),
            &auth_claims,
            &jwt::EncodingKey::from_secret("secret".as_ref()),
        )
        .unwrap();

        let _m_user_me = mock("GET", "/user/me")
            .with_status(200)
            .with_header("Content-Type", "application/json")
            .with_header("User-Agent", "CDS/sdk")
            .with_header("X-Requested-With", "X-CDS-SDK")
            .with_header("Authorization", &format!("Bearer {}", token))
            .with_body(r#"{"username": "me", "fullname": "me", "created": "2019-10-11T10:23:33.207715+02:00", "id": "xxx", "ring": "user"}"#)
            .create();

        let my_client = Client::new(&mockito::server_url(), &String::from("xxx"));
        {
            let session_token = my_client.session_token.read();
            session_token.replace(token);
        }

        let me = task::block_on(my_client.me()).unwrap();

        assert_eq!(me.username, String::from("me"));
        assert_eq!(me.fullname, String::from("me"));
        assert_eq!(me.ring, String::from("user"));
    }
}
