# mirrors
Collection of all kinds of mirrors

# Get started
Install it via: `brew install linuxsuren/linuxsuren/mp`

or via [hd](https://github.com/LinuxSuRen/http-downloader):

```shell
hd i mp
```

Pull image from mirror: `mp pull gcr.io/gitpod-io/ws-scheduler:v0.4.0`

# Run as Kubernetes Operator
Install it via the following command:

```shell
kubectl apply -f https://github.com/LinuxSuRen/mirrors/releases/latest/download/controller.yaml
```

# TODO
There are a couple of things need to be improved:

* [ ] [containerd](https://containerd.io/) support
* [ ] an easier way to submit mirror config item

## Contribution
Please run the following command before you try to create a PR:

```shell
make pre-commit
```

## Mirror of GitHub
* https://hub.fastgit.org/
* https://github.com.cnpmjs.org/
