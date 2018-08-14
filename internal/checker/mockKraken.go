package checker

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/CanalTP/gormungandr"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
)

// return the tag of the docker image to used, by default "dev" images are used
// this can be override by settings the environement varriable "GORMUNGANDR_DOCKERTEST_TAG"
func getTag() string {
	tag := os.Getenv("GORMUNGANDR_DOCKERTEST_TAG")
	if tag != "" {
		return tag
	}
	return "dev"
}

// MockManager handle the creation of kraken mock
// at the end of the test the manager must be close with Close() to release
// the ressources allocated, typically the container.
type MockManager struct {
	pool      *dockertest.Pool
	resources []*dockertest.Resource
}

func NewMockManager() (*MockManager, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, err
	}
	pool.MaxWait = 30 * time.Second
	return &MockManager{
		pool: pool,
	}, nil
}

func (m *MockManager) Close() error {
	for _, resource := range m.resources {
		if err := m.pool.Purge(resource); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockManager) DepartureBoardTest() (*gormungandr.Kraken, error) {
	return m.startKraken("departure_board_test")
}

func (m *MockManager) MainRoutingTest() (*gormungandr.Kraken, error) {
	return m.startKraken("main_routing_test")
}

func (m *MockManager) startKraken(binary string) (*gormungandr.Kraken, error) {
	if err := m.pool.Client.PullImage(docker.PullImageOptions{
		Repository:   "navitia/mock-kraken",
		Tag:          getTag(),
		OutputStream: os.Stdout,
	}, docker.AuthConfiguration{}); err != nil {
		return nil, err
	}

	options := dockertest.RunOptions{
		Repository: "navitia/mock-kraken",
		Tag:        getTag(),
		Env:        []string{"KRAKEN_GENERAL_log_level=DEBUG"},
		Cmd:        []string{fmt.Sprint("./", binary), "--GENERAL.zmq_socket", "tcp://*:30000"},
	}
	resource, err := m.pool.RunWithOptions(&options)
	m.resources = append(m.resources, resource)
	if err != nil {
		return nil, err
	}
	conStr := ""
	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err = m.pool.Retry(func() error {
		var err2 error
		var conn net.Conn
		conStr = fmt.Sprintf("localhost:%s", resource.GetPort("30000/tcp"))
		conn, err2 = net.Dial("tcp", conStr)
		if err2 != nil {
			return err2
		}
		return conn.Close()
	}); err != nil {
		return nil, err
	}
	return gormungandr.NewKraken(binary, fmt.Sprint("tcp://", conStr), 1*time.Second), nil
}
