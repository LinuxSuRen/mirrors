package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cmd/ctr/commands"
	"github.com/containerd/containerd/cmd/ctr/commands/content"
	"github.com/containerd/containerd/defaults"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/platforms"
	ver "github.com/linuxsuren/cobra-extension/version"
	"github.com/opencontainers/image-spec/identity"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"github.com/urfave/cli"
)

type MirrorOption struct {
	ConfigURL  string
	containerd bool
}

func NewMirrorPullCmd() (cmd *cobra.Command) {
	configURL := "https://raw.githubusercontent.com/LinuxSuRen/mirrors/master/images.json"

	cmd = &cobra.Command{
		Use: "mp",
	}

	opt := MirrorOption{}
	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull image from the mirror",
		Example: `mp pull gcr.io/gitpod-io/ws-scheduler:v0.4.0

mp pull registry.k8s.io/nfd/node-feature-discovery:v0.10.1 10.121.218.184:30002

Please create a pull request to submit it if there's no mirror for your desired image.'`,
		RunE: opt.runE,
	}

	flags := pullCmd.Flags()
	flags.StringVarP(&opt.ConfigURL, "config", "c", configURL, "The mirror config file path")
	flags.BoolVarP(&opt.containerd, "containerd", "", false, "The container runtime is containerd")

	// add version command
	cmd.AddCommand(ver.NewVersionCmd("linuxsuren", "mirrors", "mp", nil),
		pullCmd)
	return
}

func (o *MirrorOption) pullInContainerd(cmd *cobra.Command, args []string) (err error) {
	// client, err := containerd.New("/run/containerd/containerd.sock")
	// if err != nil {
	// 	return err
	// }
	// defer client.Close()

	// ctx := namespaces.WithNamespace(cmd.Context(), "example")
	// var image containerd.Image
	// image, err = client.Pull(ctx, args[0], containerd.WithPullUnpack, func(c *containerd.Client, rc *containerd.RemoteContext) error {
	// 	fmt.Println(".")
	// 	return nil
	// })
	// cmd.Println(image)

	clicmd := cli.Command{
		Name: "test",
		Flags: append(commands.RegistryFlags,
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
			}),
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
			return nil
		},
	}
	app := cli.NewApp()
	app.Commands = []cli.Command{clicmd}
	err = app.Run(args)
	fmt.Println("containerd")
	return
}

func (o *MirrorOption) runE(cmd *cobra.Command, args []string) (err error) {
	if len(args) <= 0 {
		return cmd.Help()
	}
	if o.containerd {
		return o.pullInContainerd(cmd, args)
	}
	if len(args) == 2 {
		return o.cacheImage(cmd, args)
	}

	var rps *http.Response
	mirrorCfg := make(map[string]string, 0)
	if rps, err = http.Get(o.ConfigURL); err == nil {
		if rps.StatusCode != http.StatusOK {
			err = fmt.Errorf("cannot get the config file, status code: %d", rps.StatusCode)
			return
		}

		var data []byte
		if data, err = ioutil.ReadAll(rps.Body); err == nil {
			if err = json.Unmarshal(data, &mirrorCfg); err != nil {
				return
			}
		}
	}

	image := args[0]
	var mirror string
	var ok bool
	if mirror, ok = mirrorCfg[image]; ok {
		imageArr := strings.Split(image, ":")
		if len(imageArr) == 2 {
			mirror = fmt.Sprintf("%s:%s", mirror, imageArr[1])
		}

		cmd.Println("found mirror:", mirror)
	} else {
		mirror = image
	}

	execCommand("docker", "pull", mirror)

	execCommand("docker", "tag", mirror, image)
	return
}

func (o *MirrorOption) cacheImage(cmd *cobra.Command, args []string) (err error) {
	src := args[0]
	target := args[1]

	target = fmt.Sprintf("%s/%s", target, src)

	execCommand("docker", "pull", src)

	execCommand("docker", "tag", src, target)

	execCommand("docker", "push", target)
	return
}

func execCommand(name string, arg ...string) (err error) {
	command := exec.Command(name, arg...)

	stdoutIn, _ := command.StdoutPipe()
	stderrIn, _ := command.StderrPipe()
	err = command.Start()
	if err != nil {
		return err
	}

	// cmd.Wait() should be called only after we finish reading
	// from stdoutIn and stderrIn.
	// wg ensures that we finish
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		_, _ = copyAndCapture(os.Stdout, stdoutIn)
		wg.Done()
	}()

	copyAndCapture(os.Stderr, stderrIn)

	wg.Wait()

	err = command.Wait()
	return
}

func copyAndCapture(w io.Writer, r io.Reader) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1024, 1024)
	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			d := buf[:n]
			out = append(out, d...)
			_, err := w.Write(d)
			if err != nil {
				return out, err
			}
		}
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			}
			return out, err
		}
	}
}
