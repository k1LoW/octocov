package cmd

import (
	"context"
	"errors"
	"strings"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/datastore"
)

func createBQTable(ctx context.Context, c *config.Config) error {
	if !c.DatastoreConfigReady() {
		return errors.New("datastore config not ready")
	}
	if err := c.BuildReportConfig(); err != nil {
		return err
	}
	datastores := []datastore.Datastore{}
	for _, s := range c.Report.Datastores {
		if !strings.HasPrefix("bq://", s) {
			continue
		}
		d, err := datastore.New(ctx, s, c.Root())
		if err != nil {
			return err
		}
		datastores = append(datastores, d)
	}

	if len(datastores) == 0 {
		return errors.New("bq:// are not exists")
	}

	for _, d := range datastores {
		b := d.(*datastore.BQ)
		if err := b.CreateTable(ctx); err != nil {
			return err
		}
	}
	return nil
}
