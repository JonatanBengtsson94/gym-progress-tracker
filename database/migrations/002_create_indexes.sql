CREATE INDEX idx_templates_user_id ON templates(user_id);
CREATE INDEX idx_workouts_template_id ON workouts(template_id);
CREATE INDEX idx_sets_workout_id ON sets(workout_id);
CREATE INDEX idx_sets_exercise_id ON sets(exercise_id);
CREATE UNIQUE INDEX idx_usernames ON users(username);
