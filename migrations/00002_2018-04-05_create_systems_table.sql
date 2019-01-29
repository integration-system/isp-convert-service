-- +goose Up
CREATE TABLE systems (
    id serial4 NOT NULL PRIMARY KEY,
    uuid UUID NOT NULL,
    "name" varchar(255) NOT NULL,
    created_at timestamp DEFAULT (now() at time zone 'utc') NOT NULL
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_created_modified_column_date()
    RETURNS TRIGGER AS
$body$
DECLARE
    modifiedExists bool;
BEGIN
    IF TG_OP = 'UPDATE' THEN
        NEW.created_at = OLD.created_at;
        NEW.modified_at = (now() at time zone 'utc');
    ELSIF TG_OP = 'INSERT' THEN
        NEW.modified_at = (now() at time zone 'utc');
    END IF;
    RETURN NEW;
END;
$body$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_created_column_date()
    RETURNS TRIGGER AS
$body$
DECLARE
    modifiedExists bool;
BEGIN
    IF TG_OP = 'UPDATE' THEN
        NEW.created_at = OLD.created_at;
    END IF;
    RETURN NEW;
END;
$body$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER update_customer_modtime BEFORE UPDATE ON systems
    FOR EACH ROW EXECUTE PROCEDURE update_created_column_date();

ALTER TABLE systems ADD CONSTRAINT "UQ_systems_uuid" UNIQUE ("uuid");


-- +goose Down
DROP TABLE systems;
