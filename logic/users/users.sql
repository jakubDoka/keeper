--+init+--
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY NOT NULL,
    email VARCHAR(320) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL DEFAULT '',
    password VARCHAR(60) NOT NULL,
)
--+create+--
INSERT INTO users (id, email, password) VALUES (%1, %2, %3)
--+set-name+--
UPDATE users SET name = %1 WHERE id = %2
--+get-password-and-id+--
SELECT password, id FROM users WHERE email = %1