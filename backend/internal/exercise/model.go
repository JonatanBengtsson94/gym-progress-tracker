package exercise

type Exercise struct {
	ExerciseId   uint32 `json:"exercise_id"`
	UserId       uint32 `json:"user_id"`
	ExerciseName string `json:"exercise_name"`
}
