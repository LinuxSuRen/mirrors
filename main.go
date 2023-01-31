package main

import (
	"fmt"
	"os"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cmd/ctr/commands"
	"github.com/containerd/containerd/cmd/ctr/commands/content"
	"github.com/containerd/containerd/defaults"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/platforms"
	"github.com/opencontainers/image-spec/identity"
	"github.com/urfave/cli"
)

func main() {
	// mp := cmd.NewMirrorPullCmd()
	// if err := mp.ExecuteContext(context.Background()); err != nil {
	// 	os.Exit(1)
	// }

	clicmd := cli.Command{
		Name:  "test",
		Flags: commands.RegistryFlags,
		Action: func(context *cli.Context) error {
			fmt.Println("here", context.Args(), context.GlobalString("address"))
			client, ctx, cancel, err := commands.NewClient(context)
			if err != nil {
				return err
			}
			defer cancel()

			ctx, done, err := client.WithLease(ctx)
			if err != nil {
				return err
			}
			defer done(ctx)

			config, err := content.NewFetchConfig(ctx, context)
			if err != nil {
				return err
			}

			img, err := content.Fetch(ctx, client, context.Args()[0], config)
			if err != nil {
				return err
			}

			var p []ocispec.Platform
			p, err = images.Platforms(ctx, client.ContentStore(), img.Target)
			if err != nil {
				return fmt.Errorf("unable to resolve image platforms: %w", err)
			}

			start := time.Now()
			for _, platform := range p {
				fmt.Printf("unpacking %s %s...\n", platforms.Format(platform), img.Target.Digest)
				i := containerd.NewImageWithPlatform(client, img, platforms.Only(platform))
				err = i.Unpack(ctx, "")
				if err != nil {
					return err
				}
				if true {
					diffIDs, err := i.RootFS(ctx)
					if err != nil {
						return err
					}
					chainID := identity.ChainID(diffIDs).String()
					fmt.Printf("image chain ID: %s\n", chainID)
				}
			}

			fmt.Printf("done: %s\t\n", time.Since(start))

			if img, err := client.ImageService().Get(ctx, context.Args()[0]); err == nil {
				img.Name = fmt.Sprintf("10.121.218.184:30002/al-cloud/test/test:%d", time.Now().UnixMicro())
				if image, err := client.ImageService().Create(ctx, img); err == nil {
					resolver, err := commands.GetResolver(ctx, context)
					ropts := []containerd.RemoteOpt{
						containerd.WithResolver(resolver),
					}

					err = client.Push(ctx, img.Name, image.Target, ropts...)
					fmt.Println("push", err, img.Name)
				} else {
					fmt.Println(err)
				}
			} else {
				fmt.Println(err)
			}

			return nil
		},
	}
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output in logs",
		},
		cli.StringFlag{
			Name:   "address, a",
			Usage:  "address for containerd's GRPC server",
			Value:  defaults.DefaultAddress,
			EnvVar: "CONTAINERD_ADDRESS",
		},
		cli.DurationFlag{
			Name:  "timeout",
			Usage: "total timeout for ctr commands",
		},
		cli.DurationFlag{
			Name:  "connect-timeout",
			Usage: "timeout for connecting to containerd",
		},
		cli.StringFlag{
			Name:   "namespace, n",
			Usage:  "namespace to use with commands",
			Value:  namespaces.Default,
			EnvVar: namespaces.NamespaceEnvVar,
		}}
	app.Commands = []cli.Command{clicmd}
	err := app.Run(os.Args)
	fmt.Println("containerd", err)
}
