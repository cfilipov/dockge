pub mod users;

use redb::{Database, TableDefinition};
use std::path::Path;

/// Table: username (string) → JSON-encoded User
pub const USERS_TABLE: TableDefinition<&str, &str> = TableDefinition::new("users");

/// Table: user_id (u64) → username (string)
pub const USERS_BY_ID_TABLE: TableDefinition<u64, &str> = TableDefinition::new("users_by_id");

/// Table: settings key (string) → value (string)
pub const SETTINGS_TABLE: TableDefinition<&str, &str> = TableDefinition::new("settings");

/// Table: image update results: "stack/service" → JSON
pub const IMAGE_UPDATES_TABLE: TableDefinition<&str, &str> = TableDefinition::new("image_updates");

/// Open (or create) the redb database and initialize tables.
pub fn open(dir: &Path) -> Result<Database, redb::Error> {
    std::fs::create_dir_all(dir).ok();
    let db_path = dir.join("dockge.redb");
    let db = Database::create(db_path)?;

    // Ensure all tables exist
    let write_txn = db.begin_write()?;
    {
        let _ = write_txn.open_table(USERS_TABLE)?;
        let _ = write_txn.open_table(USERS_BY_ID_TABLE)?;
        let _ = write_txn.open_table(SETTINGS_TABLE)?;
        let _ = write_txn.open_table(IMAGE_UPDATES_TABLE)?;
    }
    write_txn.commit()?;

    Ok(db)
}
