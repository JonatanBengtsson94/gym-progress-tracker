-- A set with zero/negative reps or a negative weight is not a fact the data
-- model should be able to represent, regardless of which endpoint writes it.

ALTER TABLE sets
  ADD CONSTRAINT sets_reps_positive CHECK (reps > 0),
  ADD CONSTRAINT sets_weight_grams_non_negative CHECK (weight_grams >= 0);
