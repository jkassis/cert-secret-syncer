package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	ex "github.com/go-cmd/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "app",
		Short: "Tool for Building and Creating Images of the App",
	}
	flagString(rootCmd, "APP", "cert-secret-syncer", "Name of the application", false)

	{
		buildCmd := &cobra.Command{
			Use:   "build",
			Short: "Builds the Docker build",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Executing build creation process")
				logCwd()
				app := viper.GetString("APP")
				version := viper.GetString("VERSION")
				host := "jkassis"
				tag := fmt.Sprintf("%s/%s:%s", host, app, version)

				doOrDie(&Opts{stdoutPrint: true}, "docker", "buildx", "build",
					// "-o", "type=local,dest=/tmp/docker-build",
					"--platform", "linux/amd64",
					"--progress=plain",
					"--build-arg", os.ExpandEnv("NPM_TOKEN=${NPM_TOKEN}"),
					"--build-arg", "APP="+app,
					"-t", tag, "-f", "./build/Dockerfile", ".")
			},
		}

		flagString(buildCmd, "VERSION", "latest", "Tag for the Docker build", false)
		rootCmd.AddCommand(buildCmd)
	}

	// pushCmd
	{
		pushCmd := &cobra.Command{
			Use:   "push",
			Short: "Pushes the latest build to the repository",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Executing push")
				logCwd()

				app := viper.GetString("APP")
				version := viper.GetString("VERSION")
				host := "jkassis"
				tag := fmt.Sprintf("%s/%s:%s", host, app, version)
				doOrDie(nil, "docker", "push", tag)
			},
		}

		flagString(pushCmd, "VERSION", "latest", "Tag for the Docker build", false)
		rootCmd.AddCommand(pushCmd)
	}

	// parse options and run the command
	viper.AutomaticEnv()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func logCwd() {
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Print(err)
		return
	}
	fmt.Println("running in " + pwd)
}

type Opts struct {
	stdoutPrint bool
	stdin       io.Reader
}

func doOrDie(opts *Opts, cmd string, args ...string) (status ex.Status, stdout, stderr io.Reader) {
	if opts == nil {
		opts = &Opts{stdoutPrint: true}
	}
	fmt.Printf("> %s %s\n", cmd, strings.Join(args, " "))
	Cmd := ex.NewCmdOptions(ex.Options{Streaming: true}, cmd, args...)
	statusChan := Cmd.StartWithStdin(opts.stdin)
	stdoutBuf := bytes.NewBuffer(nil)
	stderrBuf := bytes.NewBuffer(nil)
	go func() {
		for line := range Cmd.Stdout {
			stdoutBuf.Write([]byte(line))
			if opts.stdoutPrint {
				fmt.Println(line)
			}
		}
	}()
	go func() {
		for line := range Cmd.Stderr {
			stderrBuf.Write([]byte(line))
			fmt.Println(line)
		}
	}()
	orDie(<-statusChan)
	status = Cmd.Status()
	return status, bytes.NewReader(stdoutBuf.Bytes()), bytes.NewReader(stderrBuf.Bytes())
}

func orDie(status ex.Status) {
	if status.Exit != 0 {
		fmt.Print(status.Stderr)
		os.Exit(1)
	}
}

func flagString(cmd *cobra.Command, name, dflt, desc string, required bool) {
	cmd.PersistentFlags().String(name, dflt, desc)
	viper.BindPFlag(name, cmd.PersistentFlags().Lookup(name))
	if required {
		cmd.MarkPersistentFlagRequired(name)
	}
}
