package outbound

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AndreeJait/go-utility/v2/containerdw"
	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/AndreeJait/homelytics-agent/config"
	"github.com/AndreeJait/homelytics-agent/domain/entity"
	portOutbound "github.com/AndreeJait/homelytics-agent/port/outbound"
	"github.com/opencontainers/runtime-spec/specs-go"
)

type containerdRuntime struct {
	wrapper   containerdw.Containerd
	raw       *containerd.Client
	namespace string
}

// NewContainerdRuntime wraps containerdw and a raw containerd client into the domain port.
func NewContainerdRuntime(cfg *config.AppConfig) (portOutbound.ContainerRuntime, func() error, error) {
	raw, err := containerd.New(cfg.Containerd.Address)
	if err != nil {
		return nil, nil, fmt.Errorf("containerd raw client: %w", err)
	}

	wrapper, err := containerdw.New(&containerdw.Config{
		Address:   cfg.Containerd.Address,
		Namespace: cfg.Containerd.Namespace,
		Timeout:   cfg.Containerd.Timeout,
	})
	if err != nil {
		raw.Close()
		return nil, nil, fmt.Errorf("containerd wrapper: %w", err)
	}

	cleanup := func() error {
		_ = wrapper.Close()
		return raw.Close()
	}
	return &containerdRuntime{wrapper: wrapper, raw: raw, namespace: cfg.Containerd.Namespace}, cleanup, nil
}


func (r *containerdRuntime) withNamespace(ctx context.Context) context.Context {
	return namespaces.WithNamespace(ctx, r.namespace)
}

func (r *containerdRuntime) Status(ctx context.Context) (*entity.RuntimeStatus, error) {
	version, err := r.wrapper.Version(ctx)
	if err != nil {
		return &entity.RuntimeStatus{Connected: false, Error: err.Error()}, nil
	}

	return &entity.RuntimeStatus{
		Connected: true,
		Version:   version.Version,
		Revision:  version.Revision,
	}, nil
}

func (r *containerdRuntime) PullImage(ctx context.Context, ref string) error {
	_, err := r.wrapper.PullImage(ctx, ref)
	return err
}

func normalizeImageRef(ref string) string {
	if strings.Contains(ref, "/") && strings.Contains(ref, ":") {
		return ref
	}
	if !strings.Contains(ref, "/") {
		ref = "library/" + ref
	}
	if !strings.Contains(ref, ":") {
		ref = ref + ":latest"
	}
	return "docker.io/" + ref
}

func (r *containerdRuntime) CreateContainer(ctx context.Context, req entity.RunWorkloadRequest) (string, error) {
	id := req.ID
	if id == "" {
		id = generateContainerID()
	}

	imageRef := normalizeImageRef(req.Image)
	ctx = r.withNamespace(ctx)

	if err := r.PullImage(ctx, imageRef); err != nil {
		return "", fmt.Errorf("pull image %q: %w", req.Image, err)
	}

	image, err := r.raw.GetImage(ctx, imageRef)
	if err != nil {
		return "", fmt.Errorf("get image %q: %w", req.Image, err)
	}

	opts := []containerd.NewContainerOpts{
		containerd.WithImage(image),
		containerd.WithNewSnapshot(id, image),
	}

	specOpts := []oci.SpecOpts{
		oci.WithImageConfig(image),
	}

	if req.HostNetwork {
		specOpts = append(specOpts, oci.WithHostNamespace(specs.NetworkNamespace))
	}

	if len(req.Command) > 0 {
		specOpts = append(specOpts, oci.WithProcessArgs(append(req.Command, req.Args...)...))
	}

	if len(req.Env) > 0 {
		env := make([]string, 0, len(req.Env))
		for k, v := range req.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		specOpts = append(specOpts, oci.WithEnv(env))
	}

	opts = append(opts, containerd.WithNewSpec(specOpts...))

	container, err := r.raw.NewContainer(ctx, id, opts...)
	if err != nil {
		return "", fmt.Errorf("create container %q: %w", id, err)
	}

	return container.ID(), nil
}

func (r *containerdRuntime) StartContainer(ctx context.Context, id string) error {
	_, err := r.wrapper.StartContainer(ctx, id)
	return err
}

func (r *containerdRuntime) StopContainer(ctx context.Context, id string) error {
	return r.wrapper.StopContainer(ctx, id, "SIGTERM", 30*time.Second)
}

func (r *containerdRuntime) DeleteContainer(ctx context.Context, id string) error {
	return r.wrapper.DeleteContainer(ctx, id)
}

func (r *containerdRuntime) ListContainers(ctx context.Context) ([]entity.Workload, error) {
	containers, err := r.wrapper.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]entity.Workload, 0, len(containers))
	for _, c := range containers {
		out = append(out, r.toWorkload(c))
	}
	return out, nil
}

func (r *containerdRuntime) ContainerStatus(ctx context.Context, id string) (*entity.Workload, error) {
	c, err := r.wrapper.LoadContainer(ctx, id)
	if err != nil {
		return nil, err
	}
	w := r.toWorkload(c)
	return &w, nil
}

func (r *containerdRuntime) toWorkload(c containerdw.Container) entity.Workload {
	return entity.Workload{
		ID:        c.ID,
		Image:     c.Image,
		Status:    string(c.Status),
		CreatedAt: time.Now().UTC(),
	}
}

func generateContainerID() string {
	return fmt.Sprintf("workload-%d", time.Now().UnixNano())
}
