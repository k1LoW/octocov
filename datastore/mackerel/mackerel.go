package mackerel

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/k1LoW/octocov/report"
	mkr "github.com/mackerelio/mackerel-client-go"
)

type Mackerel struct {
	client  *mkr.Client
	service string
}

func New(client *mkr.Client, service string) (*Mackerel, error) {
	return &Mackerel{
		client:  client,
		service: service,
	}, nil
}

func (m *Mackerel) StoreReport(ctx context.Context, r *report.Report) error {
	sn := m.service
	sl, err := m.client.FindServices()
	if err != nil {
		return err
	}
	if !serviceExist(sl, sn) {
		if _, err := m.client.CreateService(&mkr.CreateServiceParam{
			Name: sn,
			Memo: "Code metrics generated by octocov",
		}); err != nil {
			return err
		}
	}
	repo := strings.ReplaceAll(r.Repository, "/", "-")
	t := r.Timestamp.Unix()
	var values []*mkr.MetricValue
	if r.Coverage != nil {
		name := fmt.Sprintf("coverage.%s", repo)
		v := &mkr.MetricValue{
			Name:  name,
			Time:  t,
			Value: r.CoveragePercent(),
		}
		values = append(values, v)
	}
	if r.CodeToTestRatio != nil {
		name := fmt.Sprintf("code-to-test-ratio.%s", repo)
		v := &mkr.MetricValue{
			Name:  name,
			Time:  t,
			Value: r.CodeToTestRatioRatio(),
		}
		values = append(values, v)
	}
	if r.TestExecutionTime != nil {
		name := fmt.Sprintf("test-execution-time.%s", repo)
		v := &mkr.MetricValue{
			Name:  name,
			Time:  t,
			Value: r.TestExecutionTimeNano() / float64(time.Second), // seconds
		}
		values = append(values, v)
	}

	if err := m.client.PostServiceMetricValues(sn, values); err != nil {
		return err
	}
	return nil
}

func (m *Mackerel) Put(ctx context.Context, path string, content []byte) error {
	return errors.New("not implemented")
}

func (m *Mackerel) FS() (fs.FS, error) {
	return nil, errors.New("not implemented")
}

func serviceExist(s []*mkr.Service, sn string) bool {
	for _, v := range s {
		if sn == v.Name {
			return true
		}
	}
	return false
}
