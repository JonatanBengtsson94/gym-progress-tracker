INSERT INTO users(user_id, username, password_hash, first_name, last_name) VALUES (1, 'Test User', '$2a$10$UYwagPueizGDPGqug3vY1uM7LMCn3yAA/xZEm.bJRwNAxzBI8i.7O', 'Test', 'User');
INSERT INTO templates(template_id, user_id, template_name) VALUES ('00000000-0000-0000-0000-000000000001', 1, 'Custom Test Template');

INSERT INTO users(user_id, username, password_hash, first_name, last_name) VALUES (2, 'Second Test User', '$2a$10$UYwagPueizGDPGqug3vY1uM7LMCn3yAA/xZEm.bJRwNAxzBI8i.7O', 'Second', 'User');

-- User 3's data is only read, never written to, so listing tests see a stable set.
INSERT INTO users(user_id, username, password_hash, first_name, last_name) VALUES (3, 'Third Test User', '$2a$10$UYwagPueizGDPGqug3vY1uM7LMCn3yAA/xZEm.bJRwNAxzBI8i.7O', 'Third', 'User');
INSERT INTO templates(template_id, user_id, template_name) VALUES ('00000000-0000-0000-0000-000000000101', 3, 'push Day');
INSERT INTO templates(template_id, user_id, template_name) VALUES ('00000000-0000-0000-0000-000000000102', 3, 'Leg Day');
INSERT INTO templates(template_id, user_id, template_name) VALUES ('00000000-0000-0000-0000-000000000103', 3, 'Pull Day');
INSERT INTO templates(template_id, user_id, template_name) VALUES ('00000000-0000-0000-0000-000000000104', 3, 'Arm Day');

-- Push Day: 202 and 203 completed at the same time, after 201; 203 has the
-- higher id, so it is the latest.
INSERT INTO workouts(workout_id, template_id, started_at, completed_at) VALUES ('00000000-0000-0000-0000-000000000201', '00000000-0000-0000-0000-000000000101', '2024-01-10 09:00:00', '2024-01-10 10:00:00');
INSERT INTO workouts(workout_id, template_id, started_at, completed_at) VALUES ('00000000-0000-0000-0000-000000000202', '00000000-0000-0000-0000-000000000101', '2024-01-20 09:00:00', '2024-01-20 10:00:00');
INSERT INTO workouts(workout_id, template_id, started_at, completed_at) VALUES ('00000000-0000-0000-0000-000000000203', '00000000-0000-0000-0000-000000000101', '2024-01-20 09:30:00', '2024-01-20 10:00:00');
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (301, (SELECT exercise_id FROM exercises WHERE user_id IS NULL AND exercise_name = 'Bench Press (Barbell)'), '00000000-0000-0000-0000-000000000201', 5, 50000);
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (302, (SELECT exercise_id FROM exercises WHERE user_id IS NULL AND exercise_name = 'Bench Press (Barbell)'), '00000000-0000-0000-0000-000000000202', 5, 55000);
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (303, (SELECT exercise_id FROM exercises WHERE user_id IS NULL AND exercise_name = 'Bench Press (Dumbbell)'), '00000000-0000-0000-0000-000000000203', 10, 20000);
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (304, (SELECT exercise_id FROM exercises WHERE user_id IS NULL AND exercise_name = 'Bench Press (Barbell)'), '00000000-0000-0000-0000-000000000203', 8, 60000);
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (305, (SELECT exercise_id FROM exercises WHERE user_id IS NULL AND exercise_name = 'Bench Press (Dumbbell)'), '00000000-0000-0000-0000-000000000203', 9, 20000);
-- Pull Day: a single workout, older than Push Day's. Leg Day and Arm Day have none.
INSERT INTO workouts(workout_id, template_id, started_at, completed_at) VALUES ('00000000-0000-0000-0000-000000000204', '00000000-0000-0000-0000-000000000103', '2024-01-05 09:00:00', '2024-01-05 10:00:00');
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (306, (SELECT exercise_id FROM exercises WHERE user_id IS NULL AND exercise_name = 'Bench Press (Barbell)'), '00000000-0000-0000-0000-000000000204', 12, 40000);

-- Explicit set ids above bypass the SERIAL sequence; resync it so the next
-- auto-generated id doesn't collide with a seeded one.
SELECT setval(pg_get_serial_sequence('sets', 'set_id'), (SELECT MAX(set_id) FROM sets));
