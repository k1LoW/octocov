package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/datastore"
	"github.com/k1LoW/octocov/datastore/bq"
)

func createBQTable(ctx context.Context, c *config.Config) error {
	if err := c.ReportConfigTargetReady(); err != nil {
		return err
	}
	datastores := map[string]datastore.Datastore{}
	for _, s := range c.Report.Datastores {
		if !strings.HasPrefix(s, "bq://") {
			continue
		}
		d, err := datastore.New(ctx, s, c.Root())
		if err != nil {
			return err
		}
		datastores[s] = d
	}

	if len(datastores) == 0 {
		return errors.New("bq:// are not exists")
	}

	for u, d := range datastores {
		b := d.(*bq.BQ)
		if err := b.CreateTable(ctx); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(os.Stderr, "%s has been created\n", u)
	}
	return nil
}
