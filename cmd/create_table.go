package cmd

import (
	"context"
	"errors"
	"strings"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/datastore"
	"github.com/k1LoW/octocov/datastore/bq"
)

func createBQTable(ctx context.Context, c *config.Config) error {
	if !c.ReportConfigDatastoresReady() {
		return errors.New("report.datastores is not set")
	}
	if err := c.BuildReportConfig(); err != nil {
		return err
	}
	datastores := []datastore.Datastore{}
	for _, s := range c.Report.Datastores {
		if !strings.HasPrefix(s, "bq://") {
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
		b := d.(*bq.BQ)
		if err := b.CreateTable(ctx); err != nil {
			return err
		}
	}
	return nil
}
