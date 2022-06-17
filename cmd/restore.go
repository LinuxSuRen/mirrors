package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
	"os"
)

type restoreOption struct {
	target string
}

func newRestoreCommand() (cmd *cobra.Command) {
	opt := &restoreOption{}
	cmd = &cobra.Command{
		Use:  "restore",
		RunE: opt.runE,
	}
	flags := cmd.Flags()
	flags.StringVarP(&opt.target, "target", "t", "ghcr.io", "")
	return
}

func (o *restoreOption) runE(cmd *cobra.Command, args []string) (err error) {
	id := getGitHubUserID()
	if id == "" {
		err = fmt.Errorf("no GitHub ID got, please check if the ENV 'GITHUB_TOKEN' exists")
		return
	}

	password := os.Getenv("GITHUB_TOKEN")

	var cf *configfile.ConfigFile
	cf, err = config.Load(os.Getenv("DOCKE_CONFIG"))
	if err != nil {
		return
	}

	creds := cf.GetCredentialsStore(o.target)
	if err := creds.Store(types.AuthConfig{
		ServerAddress: o.target,
		Username:      id,
		Password:      password,
	}); err != nil {
		return err
	}

	if err := cf.Save(); err != nil {
		return err
	}

	return
}

func getGitHubUserID() (id string) {
	token := os.Getenv("GITHUB_TOKEN")

	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	var userData map[string]interface{}
	if err := json.Unmarshal(data, &userData); err != nil {
		fmt.Println(err)
		return
	}

	id = userData["login"].(string)
	return
}
