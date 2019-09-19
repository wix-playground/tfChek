package launcher

import (
	"io"
	"log"
	"os/exec"
)

func Exmpl(writer io.WriteCloser) {
	defer writer.Close()
	cmd := exec.Command("cat", "/Users/maksymsh/terraform/production_42/run.sh")
	log.Printf("Running command and waiting for it to finish...")
	cmd.Stdout = writer
	err := cmd.Run()
	log.Printf("Command finished with error: %v", err)
}
