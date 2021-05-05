package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func App() {

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		fmt.Println("Exiting, received termination signal")
		os.Exit(1)
	}()

	for true {

		/*
			1. Scan through a list of tokens
			2. For each token note the expiration
			3. if the expiration is less than x, do the following:
				a. call the vault pkg to request a new token
				b. call the circleci package to place that new token in its env var
			4.
		*/
		fmt.Println("I ran a loop!")
		time.Sleep(5 * time.Second)
	}

}
