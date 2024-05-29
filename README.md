# Example code to get secret from kubernetes cluster

Useful where there is a resource that isn't managed by Pulumi.

It assumes you have your kubeconfig in `~/.kube/config` and that you have access to the bootstrap secret in the `kube-system` namespace

## To run

1. Clone repo: `git clone https://github.com/pierskarsenbarg/k8s-get-secret`
1. Change directory: `cd k8s-get-secret`
1. Install modules: `go mod tidy`
1. Run: `go run main.go`

## Destroy

Don't forget to tear down when you're finished:

- `go run main.go destroy`