use redb::{Database, ReadableTable, ReadableTableMetadata};
use serde::{Deserialize, Serialize};
use std::sync::Arc;

use super::{USERS_BY_ID_TABLE, USERS_TABLE};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct User {
    pub id: i32,
    pub username: String,
    pub password: String,
    pub active: bool,
}

pub struct UserStore {
    db: Arc<Database>,
}

impl UserStore {
    pub fn new(db: Arc<Database>) -> Self {
        Self { db }
    }

    pub fn find_by_username(&self, username: &str) -> Result<Option<User>, redb::Error> {
        let read_txn = self.db.begin_read()?;
        let table = read_txn.open_table(USERS_TABLE)?;
        match table.get(username)? {
            Some(val) => {
                let user: User = serde_json::from_str(val.value()).unwrap();
                if user.active {
                    Ok(Some(user))
                } else {
                    Ok(None)
                }
            }
            None => Ok(None),
        }
    }

    pub fn find_by_id(&self, id: i32) -> Result<Option<User>, redb::Error> {
        let read_txn = self.db.begin_read()?;
        let id_table = read_txn.open_table(USERS_BY_ID_TABLE)?;
        let username = match id_table.get(id as u64)? {
            Some(val) => val.value().to_string(),
            None => return Ok(None),
        };
        let users_table = read_txn.open_table(USERS_TABLE)?;
        match users_table.get(username.as_str())? {
            Some(val) => {
                let user: User = serde_json::from_str(val.value()).unwrap();
                Ok(Some(user))
            }
            None => Ok(None),
        }
    }

    pub fn count(&self) -> Result<usize, redb::Error> {
        let read_txn = self.db.begin_read()?;
        let table = read_txn.open_table(USERS_TABLE)?;
        Ok(table.len()? as usize)
    }

    pub fn create(&self, username: &str, password: &str) -> Result<User, Box<dyn std::error::Error + Send + Sync>> {
        let hash = bcrypt::hash(password, 10)?;
        self.create_with_hash(username, hash)
    }

    /// Create a user with a pre-computed password hash. Use with
    /// `hash_password_async` to avoid blocking the tokio runtime.
    pub fn create_with_hash(&self, username: &str, password_hash: String) -> Result<User, Box<dyn std::error::Error + Send + Sync>> {
        let write_txn = self.db.begin_write()?;
        let user = {
            let mut id_table = write_txn.open_table(USERS_BY_ID_TABLE)?;

            // Get next ID: find max existing key + 1
            let next_id = match id_table.last()? {
                Some(entry) => entry.0.value() + 1,
                None => 1,
            };

            let user = User {
                id: next_id as i32,
                username: username.to_string(),
                password: password_hash,
                active: true,
            };

            let json = serde_json::to_string(&user)?;
            let mut users_table = write_txn.open_table(USERS_TABLE)?;
            users_table.insert(username, json.as_str())?;
            id_table.insert(next_id, username)?;

            user
        };
        write_txn.commit()?;

        Ok(user)
    }

    pub fn delete_all(&self) -> Result<(), redb::Error> {
        let write_txn = self.db.begin_write()?;
        {
            // Drain users table
            let mut users_table = write_txn.open_table(USERS_TABLE)?;
            let user_keys: Vec<String> = users_table
                .iter()?
                .filter_map(|entry| entry.ok().map(|(k, _)| k.value().to_string()))
                .collect();
            for key in &user_keys {
                users_table.remove(key.as_str())?;
            }
        }
        {
            // Drain users_by_id table
            let mut id_table = write_txn.open_table(USERS_BY_ID_TABLE)?;
            let id_keys: Vec<u64> = id_table
                .iter()?
                .filter_map(|entry| entry.ok().map(|(k, _)| k.value()))
                .collect();
            for key in &id_keys {
                id_table.remove(*key)?;
            }
        }
        write_txn.commit()?;
        Ok(())
    }

    /// Change password with a pre-computed hash. Use with
    /// `hash_password_async` to avoid blocking the tokio runtime.
    pub fn change_password_with_hash(&self, user_id: i32, password_hash: String) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let write_txn = self.db.begin_write()?;
        {
            let id_table = write_txn.open_table(USERS_BY_ID_TABLE)?;
            let username = id_table
                .get(user_id as u64)?
                .ok_or("user not found")?
                .value()
                .to_string();

            let mut users_table = write_txn.open_table(USERS_TABLE)?;
            let json_str = users_table
                .get(username.as_str())?
                .ok_or("user not found")?
                .value()
                .to_string();

            let mut user: User = serde_json::from_str(&json_str)?;
            user.password = password_hash;
            let new_json = serde_json::to_string(&user)?;
            users_table.insert(username.as_str(), new_json.as_str())?;
        }
        write_txn.commit()?;
        Ok(())
    }
}

/// Verify a plaintext password against a bcrypt hash (sync — test-only).
#[cfg(test)]
pub fn verify_password(password: &str, hash: &str) -> bool {
    bcrypt::verify(password, hash).unwrap_or(false)
}

/// Async variant of `verify_password` — offloads bcrypt (~225ms at cost 10)
/// to a blocking thread so it doesn't starve the tokio runtime.
pub async fn verify_password_async(password: &str, hash: &str) -> bool {
    let password = password.to_string();
    let hash = hash.to_string();
    tokio::task::spawn_blocking(move || bcrypt::verify(&password, &hash).unwrap_or(false))
        .await
        .unwrap_or(false)
}

/// Async bcrypt hash — offloads to a blocking thread.
pub async fn hash_password_async(password: &str) -> Result<String, bcrypt::BcryptError> {
    let password = password.to_string();
    tokio::task::spawn_blocking(move || bcrypt::hash(&password, 10))
        .await
        .unwrap_or(Err(bcrypt::BcryptError::CostNotAllowed(0)))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn verify_correct_password() {
        let hash = bcrypt::hash("testpass123", 4).unwrap();
        assert!(verify_password("testpass123", &hash));
    }

    #[test]
    fn verify_wrong_password() {
        let hash = bcrypt::hash("testpass123", 4).unwrap();
        assert!(!verify_password("wrongpass", &hash));
    }

    #[test]
    fn verify_invalid_hash_no_panic() {
        assert!(!verify_password("password", "not-a-valid-hash"));
        assert!(!verify_password("password", ""));
    }
}
