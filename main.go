package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	b64 "encoding/base64"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	pmetav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func createPulumiProgram(secretValue string) pulumi.RunFunc {

	encodedSecretValue := b64.StdEncoding.EncodeToString([]byte(secretValue))

	return func(ctx *pulumi.Context) error {
		provider, err := kubernetes.NewProvider(ctx, "k8s-provider", &kubernetes.ProviderArgs{
			Kubeconfig: pulumi.String("~/.kube/config"),
		})
		if err != nil {
			return err
		}

		namespace, err := corev1.NewNamespace(ctx, "mynamespace", &corev1.NamespaceArgs{
			Metadata: &pmetav1.ObjectMetaArgs{
				Name: pulumi.String("mynamespace"),
			},
		}, pulumi.Provider(provider))
		if err != nil {
			return err
		}
		
		secret, err := corev1.NewSecret(ctx, "mysecret", &corev1.SecretArgs{
			Metadata: &pmetav1.ObjectMetaArgs{
				Namespace: namespace.Metadata.Name(),
			},
			Data: pulumi.StringMap{
				"mysecret": pulumi.String(encodedSecretValue),
			},
			Type: pulumi.String("Opaque"),
		}, pulumi.Provider(provider))

		ctx.Export("secretName", secret.Metadata.Name())
		ctx.Export("namespace", namespace.Metadata.Name())

		return nil
	}
}

func getSecretFromNamespace(ctx context.Context, secretName string, namespace string) (string, error) {
	home := homedir.HomeDir()
	kubeConfigPath := filepath.Join(home, ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return "", err
	}

	clientSet, err := k8s.NewForConfig(config)
	if err != nil {
		return "", err
	}

	secrets, err := clientSet.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	var secretString string

	for _, secret := range secrets.Items {
		if (strings.HasPrefix(secret.Name, secretName)) {
			secretData := secret.Data["token-secret"]
			secretString = string(secretData)
			break;
		}
	}

	return secretString, nil
}

func main() {
	destroy := false
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) > 0 {
		if argsWithoutProg[0] == "destroy" {
			destroy = true
		}
	}

	ctx := context.Background()

	projectName := "k8s-secret"
	stackName := "dev"

	secretString, err := getSecretFromNamespace(ctx, "bootstrap-", "kube-system")
	if err != nil {
		fmt.Println("problem getting secret: %v", err)
		os.Exit(1)
	}

	pulumiProgram := createPulumiProgram(secretString)

	stack, err := auto.UpsertStackInlineSource(ctx, stackName, projectName, pulumiProgram)
	if err != nil {
		fmt.Printf("Failed to set up a workspace: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Starting refresh")

	_, err = stack.Refresh(ctx)
	if err != nil {
		fmt.Printf("Failed to refresh stack: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Refresh succeeded!")

	if destroy {
		fmt.Println("Starting stack destroy")

		// wire up our destroy to stream progress to stdout
		stdoutStreamer := optdestroy.ProgressStreams(os.Stdout)

		// destroy our stack and exit early
		_, err := stack.Destroy(ctx, stdoutStreamer)
		if err != nil {
			fmt.Printf("Failed to destroy stack: %v", err)
		}
		fmt.Println("Stack successfully destroyed")
		os.Exit(0)
	}

	_, err = stack.Up(ctx)
	if err != nil {
		fmt.Printf("Failed to update stack: %v\n\n", err)
		os.Exit(1)
	}

	fmt.Println("Update succeeded!")
	
}
