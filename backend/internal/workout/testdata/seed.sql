INSERT INTO users(user_id, username, password_hash, first_name, last_name) VALUES (1, 'Test User', '$2a$10$UYwagPueizGDPGqug3vY1uM7LMCn3yAA/xZEm.bJRwNAxzBI8i.7O', 'Test', 'User');
INSERT INTO users(user_id, username, password_hash, first_name, last_name) VALUES (2, 'Second Test User', '$2a$10$UYwagPueizGDPGqug3vY1uM7LMCn3yAA/xZEm.bJRwNAxzBI8i.7O', 'Second', 'User');

INSERT INTO templates(template_id, user_id, template_name) VALUES ('00000000-0000-0000-0000-000000000001', 1, 'Push Day');
INSERT INTO templates(template_id, user_id, template_name) VALUES ('00000000-0000-0000-0000-000000000002', 2, 'Pull Day');

-- Workout 1: belongs to user 1, has two sets.
INSERT INTO workouts(workout_id, template_id, started_at, completed_at) VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '2024-01-15 09:00:00', '2024-01-15 10:00:00');
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (1, (SELECT exercise_id FROM exercises WHERE user_id IS NULL AND exercise_name = 'Bench Press (Barbell)'), '00000000-0000-0000-0000-000000000001', 8, 60000);
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (2, (SELECT exercise_id FROM exercises WHERE user_id IS NULL AND exercise_name = 'Bench Press (Dumbbell)'), '00000000-0000-0000-0000-000000000001', 5, 100000);

-- Workout 2: belongs to user 2, has one set.
INSERT INTO workouts(workout_id, template_id, started_at, completed_at) VALUES ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000002', '2024-01-16 10:00:00', '2024-01-16 11:00:00');
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (3, (SELECT exercise_id FROM exercises WHERE user_id IS NULL AND exercise_name = 'Incline Bench Press (Barbell)'), '00000000-0000-0000-0000-000000000002', 5, 120000);

-- Workout 3: belongs to user 1, has no sets (should be treated as not found).
INSERT INTO workouts(workout_id, template_id, started_at, completed_at) VALUES ('00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', '2024-01-17 08:00:00', '2024-01-17 09:00:00');

-- A custom exercise owned by user 2, so user 1 must not be able to log sets for it.
INSERT INTO exercises(exercise_id, user_id, exercise_name) VALUES ('00000000-0000-0000-0000-000000000100', 2, 'Second User Curl');

-- Explicit set ids above bypass the SERIAL sequence; resync it so the next
-- auto-generated id doesn't collide with a seeded one.
SELECT setval(pg_get_serial_sequence('sets', 'set_id'), (SELECT MAX(set_id) FROM sets));
