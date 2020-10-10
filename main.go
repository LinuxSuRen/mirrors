package main

import (
	"github.com/linuxsuren/mirrors/cmd"
	"os"
)

func main()  {
	mp := cmd.NewMirrorPullCmd()
	if err := mp.Execute(); err != nil {
		os.Exit(1)
	}
}
