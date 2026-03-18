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
                password: hash,
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

    pub fn change_password(&self, user_id: i32, new_password: &str) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let hash = bcrypt::hash(new_password, 10)?;

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
            user.password = hash;
            let new_json = serde_json::to_string(&user)?;
            users_table.insert(username.as_str(), new_json.as_str())?;
        }
        write_txn.commit()?;
        Ok(())
    }
}

/// Verify a plaintext password against a bcrypt hash.
pub fn verify_password(password: &str, hash: &str) -> bool {
    bcrypt::verify(password, hash).unwrap_or(false)
}
