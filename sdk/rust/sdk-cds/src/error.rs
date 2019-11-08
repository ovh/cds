use http::method::InvalidMethod;
use std::convert::{From, Into};
use std::error::Error as StdError;
use std::fmt;
use std::io::Error as IOError;

use jwt;
use reqwest;

macro_rules! from_error {
    ($type:ty, $target:ident, $targetvar:expr) => {
        impl From<$type> for $target {
            fn from(s: $type) -> Self {
                $targetvar(s.into())
            }
        }
    };
}
macro_rules! from_error_str {
    ($type:ty, $target:ident, $targetvar:expr) => {
        impl From<$type> for $target {
            fn from(s: $type) -> Self {
                $targetvar(s.description().to_string())
            }
        }
    };
}

#[derive(Serialize, Deserialize, Fail, Debug)]
/// All errors for CDS SDK
pub enum Error {
    #[fail(display = "API Error: {:?}", _0)]
    ApiError(ApiError),
    #[fail(display = "IO Error: {:?}", _0)]
    IoError(String),
    #[fail(display = "reqwest Error: {:?}", _0)]
    ReqwestError(String),
    #[fail(display = "invalid method: {:?}", _0)]
    InvalidMethod(String),
    #[fail(display = "jwt error: {:?}", _0)]
    JWTError(String),
    #[fail(display = "json error: {:?}", _0)]
    JsonError(String),
    #[fail(display = "custom error: {:?}", _0)]
    Custom(String),
}

#[derive(Serialize, Deserialize, Default)]
#[serde(default)]
/// Error from CDS API
pub struct ApiError {
    pub status: u16,
    pub message: String,
    pub uuid: String,
}

from_error_str!(IOError, Error, Error::IoError);
from_error_str!(reqwest::Error, Error, Error::ReqwestError);
from_error_str!(InvalidMethod, Error, Error::InvalidMethod);
from_error!(String, Error, Error::Custom);
from_error!(ApiError, Error, Error::ApiError);
from_error_str!(serde_json::error::Error, Error, Error::JsonError);
from_error_str!(jwt::errors::Error, Error, Error::JWTError);
from_error_str!(
    Box<dyn std::error::Error + std::marker::Send + std::marker::Sync>,
    Error,
    Error::Custom
);

impl fmt::Display for ApiError {
    fn fmt(&self, fmt: &mut fmt::Formatter) -> fmt::Result {
        if self.status == 0 {
            write!(fmt, "message: {}, uuid: {}", self.message, self.uuid)
        } else {
            write!(
                fmt,
                "status: {}, message: {}, uuid: {}",
                self.status, self.message, self.uuid
            )
        }
    }
}

impl fmt::Debug for ApiError {
    fn fmt(&self, fmt: &mut fmt::Formatter) -> fmt::Result {
        if self.status == 0 {
            write!(fmt, "message: {}, uuid: {}", self.message, self.uuid)
        } else {
            write!(
                fmt,
                "status: {}, message: {}, uuid: {}",
                self.status, self.message, self.uuid
            )
        }
    }
}

impl ApiError {
    pub fn new<T: Into<String>>(status: u16, message: T) -> Self {
        ApiError {
            status,
            message: message.into(),
            ..Default::default()
        }
    }
}
