import { Knex } from "knex";

export async function up(knex: Knex): Promise<void> {
    return knex.schema.createTable("image_update_cache", (table) => {
        table.increments("id");
        table.string("stack_name", 255).notNullable();
        table.string("service_name", 255).notNullable();
        table.string("image_reference", 500);
        table.string("local_digest", 500).nullable();
        table.string("remote_digest", 500).nullable();
        table.boolean("has_update").defaultTo(false);
        table.integer("last_checked").nullable();
        table.unique(["stack_name", "service_name"]);
    });
}

export async function down(knex: Knex): Promise<void> {
    return knex.schema.dropTable("image_update_cache");
}
