package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/bindernews/taki/pkg/imager"
	"github.com/spf13/cobra"
)

var (
	kubectlCmd      string
	targetPod       string
	targetContainer string
	imagePath       string
)

func init() {
	rootCmd.Flags().StringVarP(&kubectlCmd, "kubectl", "k", "kubectl",
		`the kubectl cli command (default: kubectl)`)
}

var rootCmd = &cobra.Command{
	Use:   "taki",
	Short: "taki - Totally Awesome Kubernetes Imager",
	Long:  `taki is a tool for creating images of running kubernetes containers, for the purposes of incident response and digital forensics`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancelFn := context.WithCancel(context.Background())
		defer cancelFn()

		localCache := &imager.ImageCache{}
		imageTask := imager.NewImager(ctx, imager.ImagerConfig{
			KubectlCmd: strings.Split(kubectlCmd, " "),
			Pod:        targetPod,
			Container:  targetContainer,
			BaseImage:  imagePath,
			MetaCache:  localCache,
		})
		if err := imageTask.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "error: '%s'", err)
			os.Exit(1)
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "There was an error while executing taki '%s'", err)
		os.Exit(1)
	}
}
