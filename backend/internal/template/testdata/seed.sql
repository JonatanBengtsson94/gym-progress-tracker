INSERT INTO users(user_id, username, password_hash, first_name, last_name) VALUES (1, 'Test User', '$2a$10$UYwagPueizGDPGqug3vY1uM7LMCn3yAA/xZEm.bJRwNAxzBI8i.7O', 'Test', 'User');
INSERT INTO templates(user_id, template_name) VALUES (1, 'Custom Test Template');

INSERT INTO users(user_id, username, password_hash, first_name, last_name) VALUES (2, 'Second Test User', '$2a$10$UYwagPueizGDPGqug3vY1uM7LMCn3yAA/xZEm.bJRwNAxzBI8i.7O', 'Second', 'User');

-- User 3's data is only read, never written to, so listing tests see a stable set.
INSERT INTO users(user_id, username, password_hash, first_name, last_name) VALUES (3, 'Third Test User', '$2a$10$UYwagPueizGDPGqug3vY1uM7LMCn3yAA/xZEm.bJRwNAxzBI8i.7O', 'Third', 'User');
INSERT INTO templates(template_id, user_id, template_name) VALUES (101, 3, 'push Day');
INSERT INTO templates(template_id, user_id, template_name) VALUES (102, 3, 'Leg Day');
INSERT INTO templates(template_id, user_id, template_name) VALUES (103, 3, 'Pull Day');
INSERT INTO templates(template_id, user_id, template_name) VALUES (104, 3, 'Arm Day');

-- Push Day: 202 and 203 completed at the same time, after 201; 203 has the
-- higher id, so it is the latest.
INSERT INTO workouts(workout_id, template_id, started_at, completed_at) VALUES (201, 101, '2024-01-10 09:00:00', '2024-01-10 10:00:00');
INSERT INTO workouts(workout_id, template_id, started_at, completed_at) VALUES (202, 101, '2024-01-20 09:00:00', '2024-01-20 10:00:00');
INSERT INTO workouts(workout_id, template_id, started_at, completed_at) VALUES (203, 101, '2024-01-20 09:30:00', '2024-01-20 10:00:00');
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (301, 1, 201, 5, 50000);
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (302, 1, 202, 5, 55000);
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (303, 2, 203, 10, 20000);
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (304, 1, 203, 8, 60000);
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (305, 2, 203, 9, 20000);
-- Pull Day: a single workout, older than Push Day's. Leg Day and Arm Day have none.
INSERT INTO workouts(workout_id, template_id, started_at, completed_at) VALUES (204, 103, '2024-01-05 09:00:00', '2024-01-05 10:00:00');
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (306, 1, 204, 12, 40000);

-- Explicit ids above bypass the SERIAL sequences; resync them so the next
-- auto-generated id doesn't collide with a seeded one.
SELECT setval(pg_get_serial_sequence('templates', 'template_id'), (SELECT MAX(template_id) FROM templates));
SELECT setval(pg_get_serial_sequence('workouts', 'workout_id'), (SELECT MAX(workout_id) FROM workouts));
SELECT setval(pg_get_serial_sequence('sets', 'set_id'), (SELECT MAX(set_id) FROM sets));
