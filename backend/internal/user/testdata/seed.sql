INSERT INTO users(user_id, username, password_hash, first_name, last_name) VALUES (1, 'Test User', 'hashed-password', 'Test', 'User');

-- Explicit ids above bypass the SERIAL sequence; resync it so users created
-- in tests don't collide with a seeded one.
SELECT setval(pg_get_serial_sequence('users', 'user_id'), (SELECT MAX(user_id) FROM users));
