-- Exercises, templates and workouts are identified by UUIDs, so clients can
-- create them offline under ids of their own. Existing rows get new ids, and
-- every reference to them follows.

BEGIN;

-- NEW IDS

ALTER TABLE exercises ADD COLUMN new_exercise_id UUID NOT NULL DEFAULT uuidv7();
ALTER TABLE templates ADD COLUMN new_template_id UUID NOT NULL DEFAULT uuidv7();
ALTER TABLE workouts ADD COLUMN new_workout_id UUID NOT NULL DEFAULT uuidv7();

ALTER TABLE workouts ADD COLUMN new_template_id UUID;
UPDATE workouts AS w
  SET new_template_id = t.new_template_id
  FROM templates AS t
  WHERE t.template_id = w.template_id;

ALTER TABLE sets ADD COLUMN new_exercise_id UUID, ADD COLUMN new_workout_id UUID;
UPDATE sets AS s
  SET new_exercise_id = e.new_exercise_id
  FROM exercises AS e
  WHERE e.exercise_id = s.exercise_id;
UPDATE sets AS s
  SET new_workout_id = w.new_workout_id
  FROM workouts AS w
  WHERE w.workout_id = s.workout_id;

-- OLD IDS

-- Dropping a column also drops the keys and indexes on it. References go
-- first, so nothing depends on a primary key by the time it is dropped.
ALTER TABLE sets DROP COLUMN exercise_id, DROP COLUMN workout_id;
ALTER TABLE workouts DROP COLUMN template_id;
ALTER TABLE workouts DROP COLUMN workout_id;
ALTER TABLE templates DROP COLUMN template_id;
ALTER TABLE exercises DROP COLUMN exercise_id;

-- EXERCISES

ALTER TABLE exercises RENAME COLUMN new_exercise_id TO exercise_id;
ALTER TABLE exercises ADD PRIMARY KEY (exercise_id);

-- TEMPLATES

ALTER TABLE templates RENAME COLUMN new_template_id TO template_id;
ALTER TABLE templates ADD PRIMARY KEY (template_id);

-- WORKOUTS

ALTER TABLE workouts RENAME COLUMN new_workout_id TO workout_id;
ALTER TABLE workouts ADD PRIMARY KEY (workout_id);

ALTER TABLE workouts RENAME COLUMN new_template_id TO template_id;
ALTER TABLE workouts
  ALTER COLUMN template_id SET NOT NULL,
  ADD FOREIGN KEY (template_id) REFERENCES templates(template_id);

CREATE INDEX idx_workouts_template_id
  ON workouts(template_id);

-- SETS

ALTER TABLE sets RENAME COLUMN new_exercise_id TO exercise_id;
ALTER TABLE sets RENAME COLUMN new_workout_id TO workout_id;
ALTER TABLE sets
  ALTER COLUMN exercise_id SET NOT NULL,
  ALTER COLUMN workout_id SET NOT NULL,
  ADD FOREIGN KEY (exercise_id) REFERENCES exercises(exercise_id),
  ADD FOREIGN KEY (workout_id) REFERENCES workouts(workout_id);

CREATE INDEX idx_sets_workout_id
  ON sets(workout_id);
CREATE INDEX idx_sets_exercise_id
  ON sets(exercise_id);

COMMIT;
