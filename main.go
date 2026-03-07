package main

import (
	"context"
	"log"
	"os"

	"github.com/liweiming-nova/open_code/cmd"
)

func main() {
	if err := cmd.Root().Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
