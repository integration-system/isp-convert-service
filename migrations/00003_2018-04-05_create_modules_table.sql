-- +goose Up
CREATE TABLE modules (
    id serial4 NOT NULL PRIMARY KEY,
    system_id int4 NOT NULL,
    "name" varchar(255) NOT NULL,
    "active" bool DEFAULT true,
    created_at timestamp DEFAULT (now() at time zone 'utc') NOT NULL,
    last_connected_at timestamp,
    last_disconnected_at timestamp
);

ALTER TABLE modules
    ADD CONSTRAINT "FK_modules_systemId_systems_id"
    FOREIGN KEY ("system_id") REFERENCES systems ("id")
    ON DELETE CASCADE ON UPDATE CASCADE;
CREATE INDEX IX_modules_systemId ON modules USING hash (system_id);

CREATE TRIGGER update_customer_modtime BEFORE UPDATE ON modules
    FOR EACH ROW EXECUTE PROCEDURE update_created_column_date();

ALTER TABLE modules ADD CONSTRAINT "UQ_modules_systemId_name" UNIQUE ("system_id", "name");

-- +goose Down
DROP TABLE modules;
