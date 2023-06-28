package imager

import (
	"context"
	"sync"
)

// Simple function to image one or more pods at once. The caller must fill in all fields of
// 'config' other than 'Pod'. Returns an array of errors returned by the imagers.
func SimpleImagePods(ctx context.Context, config ImagerConfig, pods []string) []error {
	errors := make([]error, len(pods))
	wait := &sync.WaitGroup{}
	wait.Add(len(pods))
	for i, pod := range pods {
		podConfig := config
		podConfig.Pod = pod
		idx := i
		go func() {
			imageTask := NewImager(ctx, config)
			imageTask.CloseUpdates()
			if err := imageTask.Start(); err != nil {
				errors[idx] = err
			}
		}()
	}
	wait.Wait()
	return errors
}
