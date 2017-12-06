package main

import "github.com/leandro-lugaresi/message-cannon/cmd"

var version = "master"
var commit = ""

func main() {
	cmd.Execute(version, commit)
}
