CREATE TABLE `users` (
                         username        TEXT NOT NULL PRIMARY KEY,
                         password        TEXT NOT NULL,
                         administrator   INTEGER NOT NULL,
                         max_time        INTEGER NOT NULL,
                         cooldown        INTEGER NOT NULL,
                         expires_at      INTEGER NOT NULL,
                         banned          INTEGER NOT NULL,
                         ban_reason      TEXT NOT NULL
);

CREATE TABLE `logs` (
                        identifier      TEXT NOT NULL PRIMARY KEY,
                        username        TEXT NOT NULL,
                        target          TEXT NOT NULL,
                        method          TEXT NOT NULL,
                        duration        TEXT NOT NULL,
                        timestamp       TEXT NOT NULL
);