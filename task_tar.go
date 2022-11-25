package taki

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync/atomic"

	"github.com/bindernews/taki/task"
)

type TarTask struct {
	*task.BaseTask
	// Ouput tar file path
	Output string
	// Root of tar file contents
	Root string
	// List of file paths to collect (relative to Root)
	Files []string
	// Size of each file
	FileSizes []int64
	// Total size of all files to collect
	totalBytes int64
	// Bytes processed
	currentBytes *int64
	// tar process
	proc *exec.Cmd
}

func (tt *TarTask) Run(ctx context.Context) task.Void {
	var err error
	var listFile *os.File
	var rdRaw io.ReadCloser
	tt.currentBytes = new(int64)

	// Sum all file sizes
	tt.totalBytes = 0
	for _, s := range tt.FileSizes {
		tt.totalBytes += s
	}

	// Write list of all files to gather
	if listFile, err = os.CreateTemp("", "tar-list-*"); err != nil {
		return tt.Fail(err)
	}
	listFilePath := listFile.Name()
	defer os.Remove(listFilePath)

	for _, file := range tt.Files {
		if _, err = fmt.Fprintln(listFile, file); err != nil {
			return tt.Fail(err)
		}
	}
	if err = listFile.Close(); err != nil {
		return tt.Fail(err)
	}

	// Dtermine the arguments to tar
	args := []string{"cvJf", tt.Output, "-C", tt.Root, "--from-file=" + listFilePath}
	tt.proc = exec.CommandContext(ctx, "tar", args...)
	// Setup the pipe so we can monitor progress
	if rdRaw, err = tt.proc.StdoutPipe(); err != nil {
		return tt.Fail(err)
	}
	rd := bufio.NewReader(rdRaw)
	// Start tar
	if err = tt.proc.Start(); err != nil {
		return tt.Fail(err)
	}
	// Build size map while waiting for tar to do things
	sizeMap := make(map[string]int64, len(tt.Files))
	for i, name := range tt.Files {
		sizeMap[name] = tt.FileSizes[i]
	}
	// Now update progress once each file is completed
	var ln string
	for {
		if ln, err = rd.ReadString('\n'); err != nil {
			break
		}
		// Determine size of added file and update progress
		fSize := sizeMap[ln]
		if fSize != 0 {
			atomic.AddInt64(tt.currentBytes, fSize)
		}
	}
	// If EOF, close safely
	if err == io.EOF {
		// Update progress to "finished"
		atomic.StoreInt64(tt.currentBytes, tt.totalBytes)
		err = nil
	} else {
		// TODO close/kill process
	}
	if err != nil {
		tt.Fail(err)
	} else {
		tt.Ok(tt.Output)
	}
	return nil
}

func (tt *TarTask) GetCurrentBytes() int64 {
	return *tt.currentBytes
}

func (tt *TarTask) GetProgress() float64 {
	return float64(*tt.currentBytes) / float64(tt.totalBytes)
}
