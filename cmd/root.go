package cmd

import (
	"encoding/json"
	"fmt"
	ver "github.com/linuxsuren/cobra-extension/version"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type MirrorOption struct {
	ConfigURL string
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
		Example: `mp gcr.io/gitpod-io/ws-scheduler:v0.4.0

Please create a pull request to submit it if there's no mirror for your desired image.'`,
		RunE: opt.runE,
	}

	flags := pullCmd.Flags()
	flags.StringVarP(&opt.ConfigURL, "config", "c", configURL, "The mirror config file path")

	// add version command
	cmd.AddCommand(ver.NewVersionCmd("linuxsuren", "mirrors", "mp", nil),
		pullCmd)
	return
}

func (o *MirrorOption) runE(cmd *cobra.Command, args []string) (err error) {
	if len(args) <= 0 {
		return cmd.Help()
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
