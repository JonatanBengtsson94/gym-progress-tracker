package set

type SetResponse struct {
	Reps        uint8  `json:"reps"`
	WeightGrams uint32 `json:"weight_grams"`
}

type ExerciseSetsResponse struct {
	ExerciseId   uint32        `json:"exercise_id"`
	ExerciseName string        `json:"exercise_name"`
	Sets         []SetResponse `json:"sets"`
}

// GroupByExercise groups a flat set list by exercise. Exercises keep the order
// they first appear in, so repeating an exercise later in the list adds to its
// existing group rather than starting a second one. The result is never nil,
// so no sets serializes as an empty array.
func GroupByExercise(sets []Set) []ExerciseSetsResponse {
	exercises := make([]ExerciseSetsResponse, 0, len(sets))
	indexByExerciseId := make(map[uint32]int, len(sets))

	for _, s := range sets {
		i, ok := indexByExerciseId[s.Exercise.ExerciseId]
		if !ok {
			i = len(exercises)
			indexByExerciseId[s.Exercise.ExerciseId] = i
			exercises = append(exercises, ExerciseSetsResponse{
				ExerciseId:   s.Exercise.ExerciseId,
				ExerciseName: s.Exercise.ExerciseName,
			})
		}
		exercises[i].Sets = append(exercises[i].Sets, SetResponse{
			Reps:        s.Reps,
			WeightGrams: s.WeightGrams,
		})
	}

	return exercises
}
