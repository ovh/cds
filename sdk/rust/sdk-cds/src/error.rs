use http::method::InvalidMethod;
use std::convert::{From, Into};
use std::error::Error as StdError;
use std::fmt;
use std::io::Error as IOError;

use reqwest;

#[derive(Serialize, Deserialize, Default)]
#[serde(default)]
pub struct Error {
    pub status: u16,
    pub message: String,
    pub uuid: String,
}

impl From<IOError> for Error {
    fn from(e: IOError) -> Error {
        Error {
            message: e.description().into(),
            ..Default::default()
        }
    }
}

impl From<reqwest::Error> for Error {
    fn from(e: reqwest::Error) -> Error {
        Error {
            message: e.description().into(),
            ..Default::default()
        }
    }
}

impl From<InvalidMethod> for Error {
    fn from(e: InvalidMethod) -> Error {
        Error {
            message: e.description().into(),
            ..Default::default()
        }
    }
}

impl From<String> for Error {
    fn from(e: String) -> Error {
        Error {
            message: e,
            ..Default::default()
        }
    }
}

impl fmt::Display for Error {
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

impl fmt::Debug for Error {
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

impl Error {
    pub fn new<T: Into<String>>(status: u16, message: T) -> Self {
        Error {
            status,
            message: message.into(),
            ..Default::default()
        }
    }
}
