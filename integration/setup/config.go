// +build k8srequired

package setup

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/apptest"
	"github.com/giantswarm/apptest/integration/env"
)

type Config struct {
	AppTest apptest.Interface
	Logger  micrologger.Logger
}

func NewConfig() (Config, error) {
	var err error

	var logger micrologger.Logger
	{
		c := micrologger.Config{}

		logger, err = micrologger.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	var appTest apptest.Interface
	{
		c := apptest.Config{
			Logger: logger,

			KubeConfigPath: env.KubeConfigPath(),
		}

		appTest, err = apptest.New(c)
		if err != nil {
			return Config{}, microerror.Mask(err)
		}
	}

	c := Config{
		AppTest: appTest,
		Logger:  logger,
	}

	return c, nil
}
