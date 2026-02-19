use std::fmt;

#[derive(Debug)]
#[allow(dead_code)]
pub enum AppError {
    Validation(String),
    Auth(String),
    NotFound(String),
    Internal(String),
    Db(sqlx::Error),
}

impl fmt::Display for AppError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            AppError::Validation(msg) => write!(f, "Validation error: {}", msg),
            AppError::Auth(msg) => write!(f, "Auth error: {}", msg),
            AppError::NotFound(msg) => write!(f, "Not found: {}", msg),
            AppError::Internal(msg) => write!(f, "Internal error: {}", msg),
            AppError::Db(e) => write!(f, "Database error: {}", e),
        }
    }
}

impl std::error::Error for AppError {}

impl From<sqlx::Error> for AppError {
    fn from(e: sqlx::Error) -> Self {
        AppError::Db(e)
    }
}

impl From<std::io::Error> for AppError {
    fn from(e: std::io::Error) -> Self {
        AppError::Internal(e.to_string())
    }
}

impl From<serde_json::Error> for AppError {
    fn from(e: serde_json::Error) -> Self {
        AppError::Internal(e.to_string())
    }
}

impl From<serde_yaml::Error> for AppError {
    fn from(e: serde_yaml::Error) -> Self {
        AppError::Validation(format!("YAML parse error: {}", e))
    }
}

pub type AppResult<T> = Result<T, AppError>;

/// Build a standard Socket.IO error callback response
pub fn error_response(msg: &str) -> serde_json::Value {
    serde_json::json!({
        "ok": false,
        "msg": msg
    })
}

/// Build an i18n error callback response
#[allow(dead_code)]
pub fn error_response_i18n(msg: &str) -> serde_json::Value {
    serde_json::json!({
        "ok": false,
        "msg": msg,
        "msgi18n": true
    })
}

/// Build a standard Socket.IO success callback response
#[allow(dead_code)]
pub fn ok_response() -> serde_json::Value {
    serde_json::json!({ "ok": true })
}

/// Build a success callback response with an i18n message
pub fn ok_response_i18n(msg: &str) -> serde_json::Value {
    serde_json::json!({
        "ok": true,
        "msg": msg,
        "msgi18n": true
    })
}
