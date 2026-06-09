package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Stupnikjs/liquid/internal/runner"
)

func ParseCmd(r *runner.Runner) {
	fmt.Printf(":>")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Println(":>")
		cmd := scanner.Text()
		cmd_split := strings.Split(cmd, " ")

		if len(cmd_split) == 1 {
			switch cmd_split[0] {
			case "stats":
				fmt.Println(r.GetStats())
			case "exit":
				os.Exit(0)
			case "routes":
				// fmt.Println(r.GetRoutes())
			default:
				fmt.Printf("Unknown command: %s", cmd_split[0])
			}
		}
	}

}
