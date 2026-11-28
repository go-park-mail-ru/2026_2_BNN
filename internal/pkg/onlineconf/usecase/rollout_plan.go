package usecase

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"bnn/internal/models"
	"bnn/internal/pkg/onlineconf"

	uuid "github.com/satori/go.uuid"
)

var _ onlineconf.RolloutPlanUsecase = (*Usecase)(nil)

func (u *Usecase) ListRolloutPlans(ctx context.Context) ([]models.OnlineConfRolloutPlan, error) {
	return u.repo.ReadRolloutPlans(ctx)
}

func (u *Usecase) GetRolloutPlan(ctx context.Context, id uuid.UUID) (models.OnlineConfRolloutPlan, error) {
	if err := validateRolloutPlanID(id); err != nil {
		return models.OnlineConfRolloutPlan{}, err
	}
	return u.repo.ReadRolloutPlan(ctx, id)
}

func (u *Usecase) CreateRolloutPlan(ctx context.Context, input onlineconf.RolloutPlanInput) (models.OnlineConfRolloutPlan, error) {
	input, err := normalizeRolloutPlanInput(input)
	if err != nil {
		return models.OnlineConfRolloutPlan{}, err
	}
	return u.repo.CreateRolloutPlan(ctx, models.OnlineConfRolloutPlan{
		ID: uuid.NewV4(), Name: input.Name, Steps: input.Steps, IntervalMin: input.IntervalMin,
	})
}

func (u *Usecase) UpdateRolloutPlan(ctx context.Context, id uuid.UUID, input onlineconf.RolloutPlanInput) (models.OnlineConfRolloutPlan, error) {
	if err := validateRolloutPlanID(id); err != nil {
		return models.OnlineConfRolloutPlan{}, err
	}
	input, err := normalizeRolloutPlanInput(input)
	if err != nil {
		return models.OnlineConfRolloutPlan{}, err
	}
	return u.repo.UpdateRolloutPlan(ctx, models.OnlineConfRolloutPlan{
		ID: id, Name: input.Name, Steps: input.Steps, IntervalMin: input.IntervalMin,
	})
}

func (u *Usecase) DeleteRolloutPlan(ctx context.Context, id uuid.UUID) error {
	if err := validateRolloutPlanID(id); err != nil {
		return err
	}
	return u.repo.DeleteRolloutPlan(ctx, id)
}

func validateRolloutPlanID(id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("%w: rollout plan id is required", models.ErrValidation)
	}
	return nil
}

func normalizeRolloutPlanInput(
	input onlineconf.RolloutPlanInput,
) (onlineconf.RolloutPlanInput, error) {
	input.Name = strings.TrimSpace(input.Name)

	nameLength := utf8.RuneCountInString(input.Name)
	if nameLength == 0 ||
		nameLength > onlineconf.RolloutPlanNameMaxLength {
		return onlineconf.RolloutPlanInput{}, fmt.Errorf(
			"%w: rollout plan name must contain between 1 and %d characters",
			models.ErrValidation,
			onlineconf.RolloutPlanNameMaxLength,
		)
	}

	maxSteps := onlineconf.RolloutPercentMax -
		onlineconf.RolloutPercentMin + 1

	if len(input.Steps) < onlineconf.RolloutPlanMinSteps ||
		len(input.Steps) > maxSteps {
		return onlineconf.RolloutPlanInput{}, fmt.Errorf(
			"%w: rollout plan must contain between %d and %d steps",
			models.ErrValidation,
			onlineconf.RolloutPlanMinSteps,
			maxSteps,
		)
	}

	for i, step := range input.Steps {
		if step < onlineconf.RolloutPercentMin ||
			step > onlineconf.RolloutPercentMax {
			return onlineconf.RolloutPlanInput{}, fmt.Errorf(
				"%w: rollout steps must be between %d and %d",
				models.ErrValidation,
				onlineconf.RolloutPercentMin,
				onlineconf.RolloutPercentMax,
			)
		}

		if i > 0 && step <= input.Steps[i-1] {
			return onlineconf.RolloutPlanInput{}, fmt.Errorf(
				"%w: rollout steps must be strictly increasing",
				models.ErrValidation,
			)
		}
	}

	if input.Steps[len(input.Steps)-1] != onlineconf.RolloutPercentMax {
		return onlineconf.RolloutPlanInput{}, fmt.Errorf(
			"%w: last rollout step must be %d",
			models.ErrValidation,
			onlineconf.RolloutPercentMax,
		)
	}

	if input.IntervalMin <= 0 {
		return onlineconf.RolloutPlanInput{}, fmt.Errorf(
			"%w: rollout interval must be positive",
			models.ErrValidation,
		)
	}

	input.Steps = append([]int32(nil), input.Steps...)

	return input, nil
}
