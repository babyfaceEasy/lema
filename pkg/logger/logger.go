package logger

import "go.uber.org/zap"

func NewLogger(appEnv string) (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	logger, err = zap.NewProduction()
	if appEnv == "dev" {
		logger, err = zap.NewDevelopment()
	}

	if err != nil {
		return nil, err
	}

	logger = logger.With(zap.String("service", "github-fetcher"), zap.String("version", "1.0.0"))

	return logger, nil
}
 