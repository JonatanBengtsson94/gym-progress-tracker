INSERT INTO users(user_id, username, password, first_name, last_name) VALUES (1, 'Test User', 'Password', 'Test', 'User');
INSERT INTO users(user_id, username, password, first_name, last_name) VALUES (2, 'Second Test User', 'Password', 'Second', 'User');

INSERT INTO templates(template_id, user_id, template_name) VALUES (1, 1, 'Push Day');
INSERT INTO templates(template_id, user_id, template_name) VALUES (2, 2, 'Pull Day');

-- Workout 1: belongs to user 1, has two sets.
INSERT INTO workouts(workout_id, template_id, completed_at) VALUES (1, 1, '2024-01-15 10:00:00');
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (1, 1, 1, 8, 60000);
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (2, 2, 1, 5, 100000);

-- Workout 2: belongs to user 2, has one set.
INSERT INTO workouts(workout_id, template_id, completed_at) VALUES (2, 2, '2024-01-16 11:00:00');
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (3, 3, 2, 5, 120000);

-- Workout 3: belongs to user 1, has no sets (should be treated as not found).
INSERT INTO workouts(workout_id, template_id, completed_at) VALUES (3, 1, '2024-01-17 09:00:00');

-- A custom exercise owned by user 2, so user 1 must not be able to log sets for it.
INSERT INTO exercises(exercise_id, user_id, exercise_name) VALUES (100, 2, 'Second User Curl');

-- Explicit ids above bypass the SERIAL sequences; resync them so the next
-- auto-generated id doesn't collide with a seeded one.
SELECT setval(pg_get_serial_sequence('templates', 'template_id'), (SELECT MAX(template_id) FROM templates));
SELECT setval(pg_get_serial_sequence('workouts', 'workout_id'), (SELECT MAX(workout_id) FROM workouts));
SELECT setval(pg_get_serial_sequence('sets', 'set_id'), (SELECT MAX(set_id) FROM sets));
SELECT setval(pg_get_serial_sequence('exercises', 'exercise_id'), (SELECT MAX(exercise_id) FROM exercises));
