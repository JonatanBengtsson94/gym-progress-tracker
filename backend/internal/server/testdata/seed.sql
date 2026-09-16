INSERT INTO users(user_id, username, password) VALUES (1, 'alice', 'secret');
INSERT INTO exercises(user_id, exercise_name) VALUES (1, 'Custom Test Exercise');

INSERT INTO users(user_id, username, password) VALUES (2, 'bob', 'secret');

INSERT INTO templates(template_id, user_id, template_name) VALUES (1, 1, 'Push Day');
INSERT INTO workouts(workout_id, template_id, completed_at) VALUES (1, 1, '2024-01-15 10:00:00');
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (1, 1, 1, 8, 60000);

-- Explicit ids above bypass the SERIAL sequences; resync them so the next
-- auto-generated id doesn't collide with a seeded one.
SELECT setval(pg_get_serial_sequence('templates', 'template_id'), (SELECT MAX(template_id) FROM templates));
SELECT setval(pg_get_serial_sequence('workouts', 'workout_id'), (SELECT MAX(workout_id) FROM workouts));
SELECT setval(pg_get_serial_sequence('sets', 'set_id'), (SELECT MAX(set_id) FROM sets));
