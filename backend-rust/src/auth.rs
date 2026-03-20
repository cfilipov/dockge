use jsonwebtoken::{decode, encode, DecodingKey, EncodingKey, Header, Validation};
use rand::Rng;
use serde::{Deserialize, Serialize};
use sha3::digest::{ExtendableOutput, Update};
use std::io::Read;
use std::sync::Mutex;
use std::collections::HashMap;
use std::time::{Duration, Instant};

use crate::db::users::User;

const JWT_EXPIRATION_DAYS: u64 = 30;
const SHAKE256_LENGTH: usize = 16; // bytes → 32 hex chars
const SECRET_ALPHABET: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";

#[derive(Debug, Serialize, Deserialize)]
pub struct JwtClaims {
    pub username: String,
    pub h: String,
    pub exp: u64,
    pub iat: u64,
}

/// Create an HS256 JWT token for the given user.
pub fn create_jwt(user: &User, secret: &str) -> Result<String, jsonwebtoken::errors::Error> {
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_secs();

    let claims = JwtClaims {
        username: user.username.clone(),
        h: shake256_hex(&user.password),
        exp: now + JWT_EXPIRATION_DAYS * 24 * 60 * 60,
        iat: now,
    };

    encode(
        &Header::default(),
        &claims,
        &EncodingKey::from_secret(secret.as_bytes()),
    )
}

/// Verify a JWT token and return claims.
pub fn verify_jwt(token: &str, secret: &str) -> Result<JwtClaims, jsonwebtoken::errors::Error> {
    let mut validation = Validation::default();
    validation.validate_exp = true;

    let token_data = decode::<JwtClaims>(
        token,
        &DecodingKey::from_secret(secret.as_bytes()),
        &validation,
    )?;

    Ok(token_data.claims)
}

/// Compute SHAKE256 of data and return the first 16 bytes as hex (32 chars).
pub fn shake256_hex(data: &str) -> String {
    if data.is_empty() {
        return String::new();
    }
    let mut hasher = sha3::Shake256::default();
    hasher.update(data.as_bytes());
    let mut reader = hasher.finalize_xof();
    let mut out = vec![0u8; SHAKE256_LENGTH];
    reader.read_exact(&mut out).unwrap();
    hex::encode(out)
}

/// Generate a cryptographically random alphanumeric secret.
pub fn gen_secret(length: usize) -> String {
    let mut rng = rand::rng();
    (0..length)
        .map(|_| {
            let idx = rng.random_range(0..SECRET_ALPHABET.len());
            SECRET_ALPHABET[idx] as char
        })
        .collect()
}

/// Simple rate limiter: max 5 attempts per 15 minutes per key.
pub struct LoginRateLimiter {
    entries: Mutex<HashMap<String, Vec<Instant>>>,
    max_attempts: usize,
    window: Duration,
}

impl LoginRateLimiter {
    pub fn new() -> Self {
        Self {
            entries: Mutex::new(HashMap::new()),
            max_attempts: 5,
            window: Duration::from_secs(15 * 60),
        }
    }

    pub fn allow(&self, key: &str) -> bool {
        let mut entries = self.entries.lock().unwrap();
        let now = Instant::now();
        let attempts = entries.entry(key.to_string()).or_default();

        // Remove expired entries
        attempts.retain(|t| now.duration_since(*t) < self.window);

        if attempts.len() >= self.max_attempts {
            return false;
        }

        attempts.push(now);
        true
    }

    pub fn reset(&self, key: &str) {
        self.entries.lock().unwrap().remove(key);
    }

    pub fn reset_all(&self) {
        self.entries.lock().unwrap().clear();
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // ── shake256_hex ────────────────────────────────────────────────────

    #[test]
    fn shake256_empty_returns_empty() {
        assert_eq!(shake256_hex(""), "");
    }

    #[test]
    fn shake256_deterministic() {
        assert_eq!(shake256_hex("hello"), shake256_hex("hello"));
    }

    #[test]
    fn shake256_correct_length() {
        let result = shake256_hex("test");
        assert_eq!(result.len(), 32);
    }

    #[test]
    fn shake256_different_inputs_differ() {
        assert_ne!(shake256_hex("a"), shake256_hex("b"));
    }

    #[test]
    fn shake256_hex_chars_only() {
        let result = shake256_hex("test");
        assert!(result.chars().all(|c| c.is_ascii_hexdigit()));
    }

    // ── gen_secret ──────────────────────────────────────────────────────

    #[test]
    fn gen_secret_correct_length() {
        assert_eq!(gen_secret(32).len(), 32);
        assert_eq!(gen_secret(0).len(), 0);
        assert_eq!(gen_secret(1).len(), 1);
    }

    #[test]
    fn gen_secret_alphanumeric_only() {
        let secret = gen_secret(100);
        assert!(secret.chars().all(|c| c.is_ascii_alphanumeric()));
    }

    #[test]
    fn gen_secret_uniqueness() {
        let a = gen_secret(32);
        let b = gen_secret(32);
        assert_ne!(a, b);
    }

    // ── LoginRateLimiter ────────────────────────────────────────────────

    #[test]
    fn rate_limiter_allows_up_to_max() {
        let limiter = LoginRateLimiter::new();
        for _ in 0..5 {
            assert!(limiter.allow("user1"));
        }
    }

    #[test]
    fn rate_limiter_blocks_after_max() {
        let limiter = LoginRateLimiter::new();
        for _ in 0..5 {
            limiter.allow("user1");
        }
        assert!(!limiter.allow("user1"));
    }

    #[test]
    fn rate_limiter_reset_clears_key() {
        let limiter = LoginRateLimiter::new();
        for _ in 0..5 {
            limiter.allow("user1");
        }
        assert!(!limiter.allow("user1"));
        limiter.reset("user1");
        assert!(limiter.allow("user1"));
    }

    #[test]
    fn rate_limiter_reset_all_clears_all() {
        let limiter = LoginRateLimiter::new();
        for _ in 0..5 {
            limiter.allow("user1");
            limiter.allow("user2");
        }
        limiter.reset_all();
        assert!(limiter.allow("user1"));
        assert!(limiter.allow("user2"));
    }

    // ── JWT roundtrip ───────────────────────────────────────────────────

    #[test]
    fn jwt_roundtrip() {
        let user = User {
            id: 1,
            username: "admin".to_string(),
            password: "hashed_pw".to_string(),
            active: true,
        };
        let secret = "test-secret-key";
        let token = create_jwt(&user, secret).unwrap();
        let claims = verify_jwt(&token, secret).unwrap();
        assert_eq!(claims.username, "admin");
        assert_eq!(claims.h, shake256_hex("hashed_pw"));
    }

    #[test]
    fn jwt_wrong_secret_fails() {
        let user = User {
            id: 1,
            username: "admin".to_string(),
            password: "pw".to_string(),
            active: true,
        };
        let token = create_jwt(&user, "secret1").unwrap();
        assert!(verify_jwt(&token, "secret2").is_err());
    }
}
