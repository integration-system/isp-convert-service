-- +goose Up
CREATE TABLE configs (
    id serial8 NOT NULL PRIMARY KEY,
    module_id int4 NOT NULL,
    "version" int4 NOT NULL,
    "active" bool DEFAULT false,
    created_at timestamp DEFAULT (now() at time zone 'utc') NOT NULL,
    updated_at timestamp DEFAULT (now() at time zone 'utc') NOT NULL,
    "data" jsonb NOT NULL DEFAULT '{}'
);

ALTER TABLE configs
    ADD CONSTRAINT "FK_configs_moduleId_modules_id"
    FOREIGN KEY ("module_id") REFERENCES modules ("id")
    ON DELETE CASCADE ON UPDATE CASCADE;

CREATE INDEX IX_configs_moduleId ON configs USING hash (module_id);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION deactivate_config()
    RETURNS TRIGGER AS
$body$
DECLARE
   last_version integer;
BEGIN
    NEW.updated_at = (now() at time zone 'utc');
    IF TG_OP = 'INSERT'
    THEN
        SELECT version INTO last_version
            FROM config_service.configs
            WHERE module_id = NEW.module_id
            ORDER BY created_at DESC
            LIMIT 1;
        IF last_version IS NOT NULL
        THEN
            NEW.version = last_version + 1;
        ELSE
            NEW.version = 1;
        END IF;
    ELSE
         NEW.version = OLD.version;
         NEW.created_at = OLD.created_at;
    END IF;

    IF NEW.active = TRUE
    THEN
        UPDATE config_service.configs
        SET active = FALSE,
        updated_at = (now() at time zone 'utc')
        WHERE active = TRUE AND NEW.id != id AND module_id = NEW.module_id;
    END IF;
    RETURN NEW;
END;
$body$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER "deactivate_config"
    BEFORE INSERT OR UPDATE ON configs
    FOR EACH ROW EXECUTE PROCEDURE deactivate_config();

-- +goose Down
DROP TABLE configs;
DROP FUNCTION deactivate_config;
