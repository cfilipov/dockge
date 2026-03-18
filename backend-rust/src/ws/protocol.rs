use serde::{Deserialize, Serialize};
use serde_json::value::RawValue;

/// Client-to-server message.
#[derive(Deserialize, Debug)]
pub struct ClientMessage {
    pub id: Option<i64>,
    pub event: String,
    #[serde(default)]
    pub args: Option<Box<RawValue>>,
}

/// Server-to-client ack (response to a request with an id).
#[derive(Serialize)]
pub struct AckMessage<T: Serialize> {
    pub id: i64,
    pub data: T,
}

/// Server-to-client push event (unsolicited).
#[derive(Serialize)]
pub struct ServerMessage<T: Serialize> {
    pub event: String,
    pub data: T,
}

/// Common ack payload for success.
#[derive(Serialize)]
pub struct OkResponse {
    pub ok: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub msg: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub token: Option<String>,
}

/// Ack payload for terminal join responses.
#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct SessionResponse {
    pub ok: bool,
    pub session_id: u16,
}

/// Common ack payload for errors.
#[derive(Serialize)]
pub struct ErrorResponse {
    pub ok: bool,
    pub msg: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub msgi18n: Option<bool>,
}

impl ErrorResponse {
    pub fn new(msg: impl Into<String>) -> Self {
        Self {
            ok: false,
            msg: msg.into(),
            msgi18n: None,
        }
    }

    pub fn i18n(msg: impl Into<String>) -> Self {
        Self {
            ok: false,
            msg: msg.into(),
            msgi18n: Some(true),
        }
    }
}
