use crate::auth;
use crate::error::{ApiError, Error as CdsError};
use crate::models;
use crate::service::Service;

use std::cell::RefCell;
use std::collections::HashMap;

use chrono::{DateTime, TimeZone, Utc};
use parking_lot::RwLock;
use regex::Regex;
use reqwest::Client as HttpClient;
use serde::de::DeserializeOwned;
use serde::Serialize;

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
    pub fn status(&self) -> Result<models::MonitoringStatus> {
        let body: Vec<u8> = vec![];
        self.stream_json("GET".to_string(), "/mon/status".to_string(), body)
    }

    pub fn config(&self) -> Result<HashMap<String, String>> {
        let body: Vec<u8> = vec![];
        self.stream_json("GET".to_string(), "/config/user".to_string(), body)
    }

    /// Get minimal information about current user
    pub fn me(&self) -> Result<models::User> {
        let body: Vec<u8> = vec![];
        self.stream_json("GET".to_string(), String::from("/user/me"), body)
    }

    /// Get the list of broadcasts
    pub fn broadcasts(&self) -> Result<Vec<models::Broadcast>> {
        let body: Vec<u8> = vec![];
        self.stream_json("GET".to_string(), "/broadcast".to_string(), body)
    }

    /// Get the list of projects
    pub fn projects(&self) -> Result<Vec<models::Project>> {
        let body: Vec<u8> = vec![];
        self.stream_json("GET".to_string(), "/project".to_string(), body)
    }

    /// Get the list of applications in a project
    pub fn applications(&self, project_key: &str) -> Result<Vec<models::Application>> {
        let body: Vec<u8> = vec![];
        self.stream_json(
            "GET".to_string(),
            format!("/project/{}/applications", project_key),
            body,
        )
    }

    /// Get the application's details given the project key and the application name
    pub fn application(
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
    }

    /// Get all the workflows inside a project given his project key
    pub fn workflows(&self, project_key: &str) -> Result<Vec<models::Workflow>> {
        let body: Vec<u8> = vec![];
        self.stream_json(
            "GET".to_string(),
            format!("/project/{}/workflows", project_key),
            body,
        )
    }

    /// Get the workflow's details given his name
    pub fn workflow(&self, project_key: &str, workflow_name: &str) -> Result<models::Workflow> {
        let body: Vec<u8> = vec![];
        self.stream_json(
            "GET".to_string(),
            format!("/project/{}/workflows/{}", project_key, workflow_name),
            body,
        )
    }

    /// Fetch the count of the queue
    pub fn queue_count(&self) -> Result<models::QueueCount> {
        let body: Vec<u8> = vec![];
        self.stream_json(
            "GET".to_string(),
            "/queue/workflows/count".to_string(),
            body,
        )
    }

    /// Get all personal bookmarks
    pub fn bookmarks(&self) -> Result<Vec<models::Bookmark>> {
        let body: Vec<u8> = vec![];
        self.stream_json("GET".to_string(), String::from("/bookmarks"), body)
    }

    /// Get last workflow run given their workflow name
    pub fn last_run(&self, project_key: &str, workflow_name: &str) -> Result<models::WorkflowRun> {
        let body: Vec<u8> = vec![];
        self.stream_json(
            "GET".to_string(),
            format!(
                "/project/{}/workflows/{}/runs/latest",
                project_key, workflow_name
            ),
            body,
        )
    }

    /// Useful to register a new service to the API
    pub fn service_register(&self, service: &Service) -> Result<String> {
        self.stream_json(
            "POST".to_string(),
            "/services/register".to_string(),
            service,
        )
        .map(|serv_resp: Service| serv_resp.hash)
    }

    fn login(
        &self,
        consumer_type: String,
        body: HashMap<String, String>,
    ) -> Result<models::AuthConsumerSigninResponse> {
        self.stream_json(
            "POST".to_string(),
            format!("/auth/consumer/{}/signin", consumer_type),
            body,
        )
    }

    fn has_valid_token(&self) -> Result<bool> {
        let session_token = self.session_token.read();
        let session_token = &*session_token.borrow();
        if session_token.is_empty() {
            return Ok(false);
        }

        let token: jwt::TokenData<auth::AuthClaims> = jwt::dangerous_unsafe_decode(session_token)?;
        let expired_at: DateTime<Utc> = Utc.timestamp(token.claims.expires_at, 0);

        if expired_at < Utc::now() {
            Ok(false)
        } else {
            Ok(true)
        }
    }

    pub fn stream_json<T: Serialize, U: DeserializeOwned>(
        &self,
        method: String,
        path: String,
        body: T,
    ) -> Result<U> {
        let url = format!("{}{}", self.host, path);
        let mut req_http = HttpClient::new()
            .request(reqwest::Method::from_bytes(method.as_bytes())?, &url)
            .header(reqwest::header::CONTENT_TYPE, "application/json")
            .header(reqwest::header::USER_AGENT, "CDS/sdk")
            .header("X-Requested-With", "X-CDS-SDK");

        let check_token = !url.contains("/auth/consumer/builtin/signin")
            && !url.contains("/auth/consumer/local/signin")
            && !url.contains("/auth/consumer/local/signup")
            && !url.contains("/auth/consumer/local/verify")
            && !url.contains("/auth/consumer/worker/signin");

        if check_token && !self.has_valid_token()? && !self.token.is_empty() {
            // Renew session
            let mut body = HashMap::new();
            body.insert(String::from("token"), self.token.clone());

            let res = self.login(String::from("builtin"), body)?;
            let session_token = self.session_token.read();
            session_token.replace(res.token);
        }
        let rx_signin_routes = Regex::new(r#"/auth/consumer/.*/signin"#).unwrap();

        //No auth on signing routes
        if url.starts_with(&self.host) && !rx_signin_routes.is_match(&url) {
            let session_token = self.session_token.read();
            // auth the request
            req_http = req_http.header(
                http::header::AUTHORIZATION,
                format!("Bearer {}", &*session_token.borrow()),
            );
        }
        let mut resp_http = req_http.json(&body).send()?;

        if resp_http.status().as_u16() > 400u16 {
            let mut err: ApiError = resp_http.json().map_err(CdsError::from)?;
            err.status = resp_http.status().as_u16();
            return Err(CdsError::ApiError(err));
        }

        resp_http.json().map_err(CdsError::from)
    }
}
