package launcher

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	WD = "WORKING_DIRECTORY"
)

//Deprecated
func Exmpl(writer io.WriteCloser) {
	defer writer.Close()
	cmd := exec.Command("cat", "/Users/maksymsh/terraform/production_42/generator/generate")
	log.Printf("Running command and waiting for it to finish...")
	cmd.Stdout = writer
	err := cmd.Run()
	log.Printf("Command finished with error: %v", err)
}

func RunCommand(writer io.WriteCloser, context context.Context, cmd ...string) error {
	if len(cmd) < 1 {
		return errors.New("Empty command received")
	}

	var cwd string
	if d, ok := context.Value(WD).(string); ok {
		cwd = d
	} else {
		d, err := os.Getwd()
		if err != nil {
			return err
		}
		cwd = d
	}
	log.Printf("Working directory: %s", cwd)
	command := exec.CommandContext(context, cmd[0], cmd[1:]...)
	command.Dir = cwd
	log.Printf("Running command and waiting for it to finish...")
	command.Stdout = writer
	command.Stderr = writer
	err := command.Run()
	log.Printf("Command finished with error: %v", err)
	return err
}

func RunCommands(writer io.WriteCloser, context context.Context, commands *map[string][]string) <-chan error {
	errc := make(chan error)
	defer close(errc)
	if commands == nil || len(*commands) == 0 {
		log.Printf("No commands were passed")
		return nil
	}
	for name, cmd := range *commands {
		log.Printf("[%s]\tLaunching command: %s", name, strings.Join(cmd, " "))
		err := RunCommand(writer, context, cmd...)
		if err != nil {
			log.Printf("Command failed. Error: %s", err)
			errc <- err
		}
	}
	return errc
}
