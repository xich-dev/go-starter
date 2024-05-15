BEGIN;

CREATE TABLE orgs (
    id              UUID        DEFAULT gen_random_uuid(),
    name            TEXT        NOT NULL,
    owner_id        UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMPTZ,

    PRIMARY KEY (id),
    UNIQUE (name)
);

CREATE TABLE users (
    id            UUID        DEFAULT gen_random_uuid(),
    name          TEXT        NOT NULL,
    phone         TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    password_salt TEXT        NOT NULL,
    refresh_token TEXT,
    org_id        UUID        NOT NULL,
    deleted_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at    TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,

    UNIQUE (phone),
    UNIQUE (name),
    PRIMARY KEY (id),
    FOREIGN KEY (org_id) REFERENCES orgs (id) ON DELETE SET NULL ON UPDATE CASCADE
);

ALTER TABLE orgs ADD CONSTRAINT orgs_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES users (id) ON DELETE SET NULL ON UPDATE CASCADE;

CREATE TABLE phone_code (
    phone      VARCHAR(64) NOT NULL,
    typ        VARCHAR(32) NOT NULL,
    code       VARCHAR(16) NOT NULL,
    used       BOOLEAN     NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expired_at TIMESTAMPTZ NOT NULL,
    
    PRIMARY KEY (phone, typ)
);


CREATE TABLE access_rules (
    id          UUID        DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMPTZ,

    PRIMARY KEY (id),
    UNIQUE (name)
);

CREATE TABLE user_access_rules (
    user_id     UUID NOT NULL,
    rule_id     UUID NOT NULL,
    PRIMARY KEY (user_id, rule_id),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (rule_id) REFERENCES access_rules (id) ON DELETE CASCADE ON UPDATE CASCADE
);


COMMIT;
