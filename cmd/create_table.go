package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"cloud.google.com/go/bigquery"
	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/datastore"
)

func createBQTable(ctx context.Context, c *config.Config) error {
	if !c.DatastoreConfigReady() {
		return errors.New("datastore config not ready")
	}
	if err := c.BuildDatastoreConfig(); err != nil {
		return err
	}
	if c.Datastore.BQ == nil {
		return errors.New("datastore.bq not ready")
	}
	_, _ = fmt.Fprintf(os.Stderr, "Creating BigQuery table: %s:%s.%s ...\n", c.Datastore.BQ.Project, c.Datastore.BQ.Dataset, c.Datastore.BQ.Table)

	client, err := bigquery.NewClient(ctx, c.Datastore.BQ.Project)
	if err != nil {
		return err
	}
	defer client.Close()
	b, err := datastore.NewBQ(client, c.Datastore.BQ.Dataset, c.Datastore.BQ.Table)
	if err != nil {
		return err
	}
	return b.CreateTable(ctx)
}
