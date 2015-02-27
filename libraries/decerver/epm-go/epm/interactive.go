package epm

import (
	"bufio"
	"fmt"
	"os"
)

func (e *EPM) Repl() {
	fmt.Println("#######################################################")
	fmt.Println("##  Welcome to the interactive EPM shell ##############")
	fmt.Println("#######################################################")

	reader := bufio.NewReader(os.Stdin)
	for {
		lines := []string{}
		fmt.Print(">>")
		for {
			text, _ := reader.ReadString('\n')
			if text == "\n" {
				break
			}
			lines = append(lines, text)
		}
		// check lines for special syntax things

		// else parse for normal cmds
		lines = lines[:len(lines)-1] // shave off trailing new line
		err := e.parse(lines)
		if err != nil {
			fmt.Println("!>> Parse error:", err)
			continue
		}
		// epm execute jobs
		e.ExecuteJobs()
		// wait for a block
		e.chain.Commit()
		// remove jobs for next run
		e.jobs = []Job{}
	}
}
