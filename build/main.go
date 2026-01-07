package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	ex "github.com/go-cmd/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	log.SetFlags(0) // Simple log format
	initializeFlags()

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "build",
	Short: "Tool for building and pushing Docker images",
	Long:  `A tool that facilitates the building and pushing of Docker images with configurable parameters.`,
}

func initializeFlags() {
	flagString(rootCmd, "IMAGE", "cert-secret-syncer", "Name of the image", false)
	flagString(rootCmd, "HOST", "default-host", "Host for the Docker registry", false)
	flagString(rootCmd, "VERSION", "latest", "Tag for the Docker image", false)
	flagString(rootCmd, "PLATFORMS", "linux/amd64", "Comma-separated target platforms for buildx", false)
	flagBool(rootCmd, "PUSH", false, "Push image after build (required for multi-arch)", false)
	rootCmd.AddCommand(buildCmd, pushCmd)
	viper.AutomaticEnv()
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Builds the Docker image",
	Long:  `Builds a Docker image based on the provided configuration flags and environment variables.`,
	RunE:  executeBuild,
}

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Pushes the Docker image",
	Long:  `Pushes the latest Docker image to the specified registry using the provided configuration.`,
	RunE:  executePush,
}

func executeBuild(cmd *cobra.Command, args []string) error {
	log.Println("Executing build process")
	logCurrentWorkingDirectory()
	image := viper.GetString("IMAGE")
	host := viper.GetString("HOST")
	version := viper.GetString("VERSION")
	platforms := viper.GetString("PLATFORMS")
	push := viper.GetBool("PUSH")
	if platforms == "" {
		platforms = "linux/amd64"
	}
	if strings.Contains(platforms, ",") && !push {
		return fmt.Errorf("multi-arch build requires PUSH=true")
	}

	tag := fmt.Sprintf("%s/%s:%s", host, image, version)
	args := []string{
		"buildx", "build",
		"--platform", platforms,
		"--progress=plain",
	}
	if push {
		args = append(args, "--push")
	} else {
		args = append(args, "--load")
	}
	args = append(args,
		"--build-arg", os.ExpandEnv("NPM_TOKEN=${NPM_TOKEN}"),
		"-t", tag, "-f", "./build/Dockerfile", ".",
	)

	_, err := executeCommand(&Opts{stdoutPrint: true}, "docker", args...)
	return err
}

func executePush(cmd *cobra.Command, args []string) error {
	log.Println("Executing push process")
	logCurrentWorkingDirectory()
	image, host, version := viper.GetString("IMAGE"), viper.GetString("HOST"), viper.GetString("VERSION")
	tag := fmt.Sprintf("%s/%s:%s", host, image, version)

	_, err := executeCommand(nil, "docker", "push", tag)
	return err
}

func logCurrentWorkingDirectory() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting current directory: %v", err)
		return
	}
	log.Println("Running in:", pwd)
}

type Opts struct {
	stdoutPrint bool
	stdin       io.Reader
}

func executeCommand(opts *Opts, cmdString string, args ...string) (ex.Status, error) {
	if opts == nil {
		opts = &Opts{stdoutPrint: true}
	}
	log.Printf("> %s %s\n", cmdString, strings.Join(args, " "))
	cmd := ex.NewCmdOptions(ex.Options{Streaming: true}, cmdString, args...)
	statusChan := cmd.StartWithStdin(opts.stdin)
	stdoutBuf, stderrBuf := bytes.NewBuffer(nil), bytes.NewBuffer(nil)

	go handleStream(cmd.Stdout, stdoutBuf, opts.stdoutPrint)
	go handleStream(cmd.Stderr, stderrBuf, true)

	status := <-statusChan
	if status.Exit != 0 {
		log.Printf("Command failed with exit code %d and error: %s", status.Exit, stderrBuf.String())
		return status, fmt.Errorf("command failed: %s", stderrBuf.String())
	}
	return status, nil
}

func handleStream(stream <-chan string, buffer *bytes.Buffer, shouldPrint bool) {
	for line := range stream {
		buffer.WriteString(line)
		if shouldPrint {
			log.Println(line)
		}
	}
}

func flagString(cmd *cobra.Command, name, defaultValue, description string, required bool) {
	cmd.PersistentFlags().String(name, defaultValue, description)
	err := viper.BindPFlag(name, cmd.PersistentFlags().Lookup(name))
	if err != nil {
		log.Fatalf("Error binding flag '%s': %v", name, err)
	}
	if required {
		err := cmd.MarkPersistentFlagRequired(name)
		if err != nil {
			log.Fatalf("Error marking flag '%s' as required: %v", name, err)
		}
	}
}

func flagBool(cmd *cobra.Command, name string, defaultValue bool, description string, required bool) {
	cmd.PersistentFlags().Bool(name, defaultValue, description)
	err := viper.BindPFlag(name, cmd.PersistentFlags().Lookup(name))
	if err != nil {
		log.Fatalf("Error binding flag '%s': %v", name, err)
	}
	if required {
		err := cmd.MarkPersistentFlagRequired(name)
		if err != nil {
			log.Fatalf("Error marking flag '%s': %v", name, err)
		}
	}
}

