package main

import "github.com/leandro-lugaresi/message-cannon/cmd"

var version = "master"

func main() {
	cmd.Execute(version)
}
