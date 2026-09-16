package template

import (
	"context"
	"fmt"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresTemplateRepository struct {
	db *pgxpool.Pool
}

func NewPostgresTemplateRepository(db *pgxpool.Pool) *PostgresTemplateRepository {
	return &PostgresTemplateRepository{db: db}
}

func (r *PostgresTemplateRepository) CreateTemplate(ctx context.Context, template Template) (Template, error) {
	query := `
		INSERT INTO templates (user_id, template_name)
		VALUES ($1, $2)
		RETURNING template_id
	`
	err := r.db.QueryRow(ctx, query, template.UserId, template.TemplateName).Scan(&template.TemplateId)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return Template{}, ErrTemplateAlreadyExists
		}
		return Template{}, fmt.Errorf("Create template failed: %w", err)
	}

	return template, nil
}
