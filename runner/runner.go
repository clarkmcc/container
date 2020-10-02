package runner

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultHostAddress = "127.0.0.1"
)

var (
	RegistryExtensionOptions = []string{".com", ".io", ".org", ".net"}
	DefaultContainerName     = uuid.New().String()
	ErrNoContainerId         = errors.New("container id does not exist")
)

// ContainerRunnerInterface describes something that can start and stop containers
type ContainerRunnerInterface interface {
	Start(context.Context) error
	Stop(context.Context) error
}

// ContainerRunner implements ContainerRunnerInterface and can construct a custom
// container with image and port options
type ContainerRunner struct {
	name         string
	image        string
	ports        []string
	env          []string
	exposedPorts nat.PortSet
	portBindings nat.PortMap
	opts         *ContainerRunnerOpts
	client       *client.Client
	// id managed by the runner itself
	id string
}

// ContainerRunnerOpts allows customization of the runner's behavior
type ContainerRunnerOpts struct {
	// If RemoveOnFinalization is enabled, the container will be removed
	// after it is stopped.
	RemoveOnFinalization bool
}

// NewContainerRunner builds a runner that can be used to start and stop
// containers using the locally installed docker engine
func NewContainerRunner() *ContainerRunner {
	return &ContainerRunner{
		exposedPorts: map[nat.Port]struct{}{},
		portBindings: map[nat.Port][]nat.PortBinding{},
		env:          []string{},
		opts: &ContainerRunnerOpts{
			RemoveOnFinalization: true,
		},
	}
}

// WithPorts sets the ports that should be exposed and forwarded from the
// container
func (r *ContainerRunner) WithPorts(ports ...int) *ContainerRunner {
	for _, p := range ports {
		port := strconv.Itoa(p)
		r.ports = append(r.ports, port)
		r.exposedPorts[nat.Port(port)] = struct{}{}
		r.portBindings[nat.Port(port)] = []nat.PortBinding{
			{
				HostIP:   DefaultHostAddress,
				HostPort: port,
			},
		}
	}
	return r
}

// WithImage sets the container image that should be used. It defaults to
// the docker registry.
func (r *ContainerRunner) WithImage(image string) *ContainerRunner {
	r.image = image
	if !substringContainedInSlice(image, RegistryExtensionOptions) {
		r.image = fmt.Sprintf("docker.io/library/%v", image)
	}
	return r
}

// WithName sets the name of the container. Note that running Start with a
// container name that already exists will cause Start to fail.
func (r *ContainerRunner) WithName(name string) *ContainerRunner {
	r.name = name
	if len(name) == 0 {
		r.name = DefaultContainerName
	}
	return r
}

func (r *ContainerRunner) WithEnvironmentVariable(key, val string) *ContainerRunner {
	r.env = append(r.env, fmt.Sprintf("%v=%v", key, val))
	return r
}

// WithOptions sets the options that the runner should run with=
func (r *ContainerRunner) WithOptions(opts *ContainerRunnerOpts) *ContainerRunner {
	r.opts = opts
	return r
}

// Start starts the container with the provided options
func (e *ContainerRunner) Start(ctx context.Context) error {
	var err error
	e.client, err = client.NewEnvClient()
	if err != nil {
		return fmt.Errorf("creating env client: %w", err)
	}

	log.Infoln("pulling image")
	_, err = e.client.ImagePull(ctx, e.image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pulling image: %w", err)
	}

	log.Infoln("creating container")
	resp, err := e.client.ContainerCreate(ctx, &container.Config{
		Image:        e.image,
		ExposedPorts: e.exposedPorts,
	}, &container.HostConfig{
		PortBindings: e.portBindings,
	}, nil, e.name)
	if err != nil {
		return fmt.Errorf("creating container: %w", err)
	}

	// Save the container id
	e.id = resp.ID

	log.Infoln("starting container")
	if err := e.client.ContainerStart(ctx, e.id, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("starting container: %w", err)
	}
	log.Infoln("container started")
	return nil
}

// Stop stops the container that was started using Start
func (e *ContainerRunner) Stop(ctx context.Context) error {
	log.Infoln("stopping container")
	// If we don't have a container id
	if len(e.id) == 0 {
		return ErrNoContainerId
	}

	timeout := time.Minute
	err := e.client.ContainerStop(ctx, e.id, &timeout)
	if err != nil {
		return fmt.Errorf("stopping container: %w", err)
	}
	log.Infoln("container stopped")
	if e.opts.RemoveOnFinalization {
		log.Infoln("removing container")
		err = e.client.ContainerRemove(ctx, e.id, types.ContainerRemoveOptions{})
		if err != nil {
			return fmt.Errorf("removing container: %w", err)
		}
		log.Infoln("container removed")
	}
	return nil
}

// substringContainedInSlice returns true if the substr can be found as a substring
// of any member of slice
func substringContainedInSlice(str string, substrs []string) bool {
	for _, s := range substrs {
		if strings.Contains(str, s) {
			return true
		}
	}
	return false
}
