package cron

import (
	"context"
	"go-timekeeper/internal/config"
	"go-timekeeper/internal/logger"
	"go-timekeeper/internal/service"

	"github.com/robfig/cron/v3"
)

const cleanupTokensCronName = "Cleanup expired and revoked tokens cron"

// InitCrons initializes the cron jobs.
func InitCrons(
	logger *logger.Logger,
	config *config.Config,
	userService service.UserServiceInterface,
) *cron.Cron {
	ctx := context.Background()
	c := cron.New()
	_, err := c.AddFunc(config.Cron.CronCleanupRevokedTokensSpec, func() {
		err := userService.CleanUp(ctx, config.Cron.CronCleanupRevokedTokensIntervalDays)
		if err != nil {
			logger.Errorf("%s cron error: %s", cleanupTokensCronName, err)
		}
		logger.Infof("%s finished.", cleanupTokensCronName)
	})
	if err != nil {
		logger.WithError(err).Errorf("failed to schedule %s", cleanupTokensCronName)
		return c
	}
	logger.Info("Crons initialized successfully")
	return c
}
