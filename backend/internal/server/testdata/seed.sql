INSERT INTO users(user_id, username, password) VALUES (1, 'alice', 'secret');
INSERT INTO exercises(user_id, exercise_name) VALUES (1, 'Custom Test Exercise');

INSERT INTO users(user_id, username, password) VALUES (2, 'bob', 'secret');

INSERT INTO templates(template_id, user_id, template_name) VALUES (1, 1, 'Push Day');
INSERT INTO workouts(workout_id, template_id, completed_at) VALUES (1, 1, '2024-01-15 10:00:00');
INSERT INTO sets(set_id, exercise_id, workout_id, reps, weight_grams) VALUES (1, 1, 1, 8, 60000);
