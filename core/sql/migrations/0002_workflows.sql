

-- +goose Up
CREATE TABLE flows (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    engine VARCHAR(50) NOT NULL,
    additional_info JSONB,
    tags TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    parameters_schema JSONB,
    code_location VARCHAR(500),
    entrypoint VARCHAR(255)
);

CREATE TRIGGER set_timestamp_flows BEFORE UPDATE ON flows FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();


-- +goose Down