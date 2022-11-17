package elasticsearchio

import (
	"context"
	"fmt"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"

	"github.com/johannaojeling/go-beam-pipeline/pkg/internal/testutils"
)

const (
	esNetwork = "es-network"
	esImage   = "elasticsearch:8.3.1"
	esPort    = "9200"
)

type Suite struct {
	suite.Suite
	ctx       context.Context
	container testcontainers.Container
	network   testcontainers.Network
	URL       string
}

func (s *Suite) SetupSuite() {
	s.ctx = context.Background()
	s.network = testutils.CreateNetwork(s.ctx, s.T(), esNetwork)

	env := map[string]string{
		"discovery.type":         "single-node",
		"xpack.security.enabled": "false",
	}
	cfg := testutils.ContainerConfig{
		Image:      esImage,
		Env:        env,
		Networks:   []string{esNetwork},
		Ports:      []string{esPort + "/tcp"},
		WaitForLog: "started",
	}

	s.container = testutils.CreateContainer(s.ctx, s.T(), cfg)

	address := testutils.GetContainerAddress(s.ctx, s.T(), s.container, esPort)
	s.URL = fmt.Sprintf("http://%s", address)

	healthURL := s.URL + "/_cluster/health"
	testutils.WaitUntilHealthy(s.ctx, s.T(), healthURL, 30*time.Second)
}

func (s *Suite) TearDownSuite() {
	testutils.TerminateContainer(s.ctx, s.T(), s.container)
	testutils.RemoveNetwork(s.ctx, s.T(), s.network)
}
