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

impl OkResponse {
    pub fn simple() -> Self {
        Self {
            ok: true,
            msg: None,
            token: None,
        }
    }
}

/// Generic wrapper for broadcast events with an `items` map.
/// Used by stacks, containers, networks, images, and volumes broadcasts.
#[derive(Serialize)]
pub struct ItemsEvent<T: Serialize> {
    pub items: T,
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

#[cfg(test)]
mod tests {
    use super::*;

    // ── OkResponse ──────────────────────────────────────────────────────

    #[test]
    fn ok_response_simple() {
        let r = OkResponse::simple();
        assert!(r.ok);
        assert!(r.msg.is_none());
        assert!(r.token.is_none());
    }

    #[test]
    fn ok_response_omits_optional_fields() {
        let r = OkResponse::simple();
        let json = serde_json::to_string(&r).unwrap();
        assert!(!json.contains("msg"));
        assert!(!json.contains("token"));
        assert!(json.contains("\"ok\":true"));
    }

    // ── ErrorResponse ───────────────────────────────────────────────────

    #[test]
    fn error_response_new() {
        let r = ErrorResponse::new("something failed");
        assert!(!r.ok);
        assert_eq!(r.msg, "something failed");
        assert!(r.msgi18n.is_none());
    }

    #[test]
    fn error_response_i18n() {
        let r = ErrorResponse::i18n("authIncorrectCreds");
        assert!(!r.ok);
        assert_eq!(r.msg, "authIncorrectCreds");
        assert_eq!(r.msgi18n, Some(true));
    }

    // ── ClientMessage deserialization ────────────────────────────────────

    #[test]
    fn client_message_from_json() {
        let json = r#"{"id":1,"event":"login","args":["admin","pass"]}"#;
        let msg: ClientMessage = serde_json::from_str(json).unwrap();
        assert_eq!(msg.id, Some(1));
        assert_eq!(msg.event, "login");
        assert!(msg.args.is_some());
    }

    #[test]
    fn client_message_no_args() {
        let json = r#"{"event":"ping"}"#;
        let msg: ClientMessage = serde_json::from_str(json).unwrap();
        assert!(msg.id.is_none());
        assert_eq!(msg.event, "ping");
        assert!(msg.args.is_none());
    }
}
