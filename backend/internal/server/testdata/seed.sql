INSERT INTO users(user_id, username, password_hash, first_name, last_name) VALUES (1, 'alice', '$2a$10$UYwagPueizGDPGqug3vY1uM7LMCn3yAA/xZEm.bJRwNAxzBI8i.7O', 'Alice', 'Anderson');
INSERT INTO exercises(user_id, exercise_name) VALUES (1, 'Custom Test Exercise');

INSERT INTO users(user_id, username, password_hash, first_name, last_name) VALUES (2, 'bob', '$2a$10$UYwagPueizGDPGqug3vY1uM7LMCn3yAA/xZEm.bJRwNAxzBI8i.7O', 'Bob', 'Brown');

INSERT INTO templates(template_id, user_id, template_name) VALUES (1, 1, 'Push Day');
INSERT INTO workouts(workout_id, template_id, started_at, completed_at) VALUES (1, 1, '2024-01-15 09:00:00', '2024-01-15 10:00:00');
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (1, 1, 1, 8, 60000);

-- carol's data is only read, never written to, so tests can assert her exact template list.
INSERT INTO users(user_id, username, password_hash, first_name, last_name) VALUES (3, 'carol', '$2a$10$UYwagPueizGDPGqug3vY1uM7LMCn3yAA/xZEm.bJRwNAxzBI8i.7O', 'Carol', 'Clark');
INSERT INTO templates(template_id, user_id, template_name) VALUES (10, 3, 'Carol Legs');
INSERT INTO templates(template_id, user_id, template_name) VALUES (11, 3, 'Carol Back');
INSERT INTO workouts(workout_id, template_id, started_at, completed_at) VALUES (10, 10, '2024-02-01 09:00:00', '2024-02-01 10:00:00');
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (10, 1, 10, 5, 80000);

-- Explicit ids above bypass the SERIAL sequences; resync them so the next
-- auto-generated id doesn't collide with a seeded one.
SELECT setval(pg_get_serial_sequence('templates', 'template_id'), (SELECT MAX(template_id) FROM templates));
SELECT setval(pg_get_serial_sequence('workouts', 'workout_id'), (SELECT MAX(workout_id) FROM workouts));
SELECT setval(pg_get_serial_sequence('sets', 'set_id'), (SELECT MAX(set_id) FROM sets));
