package deploy

import (
	"context"

	"github.com/gophertribe/pussh"
)

type PushDockerOpts struct {
	SSHKeyPath string
	Platform   string
}

func PushDocker(ctx context.Context, image, target string, opts PushDockerOpts) error {
	runnerOpts := pussh.RunnerOptions{
		Image:             image,
		SSHAddress:        target,
		SSHKeyPath:        opts.SSHKeyPath,
		Platform:          opts.Platform,
		ImageTransferMode: pussh.TransferModeSCP,
	}
	return pussh.Execute(ctx, runnerOpts)
}
