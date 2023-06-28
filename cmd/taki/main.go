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
	targetPods      []string
	targetContainer string
	imagePath       string
)

func init() {
	rootCmd.Flags().StringVarP(&kubectlCmd, "kubectl", "k", "kubectl",
		`the kubectl cli command (default: kubectl)`)
	rootCmd.Flags().StringArrayVarP(&targetPods, "pod", "p", []string{},
		`the pod(s) to debug, may be given multiple times`)
	rootCmd.MarkFlagRequired("pod")
	rootCmd.Flags().StringVarP(&targetContainer, "container", "c", "",
		`the container to image in the given pod(s)`)
	rootCmd.MarkFlagRequired("container")
	rootCmd.Flags().StringVar(&imagePath, "image", "",
		`local path to base image to compare against`)
	rootCmd.MarkFlagRequired("image")
}

var rootCmd = &cobra.Command{
	Use:   "taki",
	Short: "taki - Totally Awesome Kubernetes Imager",
	Long:  `taki is a tool for creating images of running kubernetes containers, for the purposes of incident response and digital forensics`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancelFn := context.WithCancel(context.Background())
		defer cancelFn()

		config := imager.ImagerConfig{
			KubectlCmd: strings.Split(kubectlCmd, " "),
			Pod:        "",
			Container:  targetContainer,
			BaseImage:  imagePath,
		}
		for i, err := range imager.SimpleImagePods(ctx, config, targetPods) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "error for pod '%s': '%s'", targetPods[i], err)
			}
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "There was an error while executing taki '%s'", err)
		os.Exit(1)
	}
}
