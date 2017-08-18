package main

import "github.com/leandro-lugaresi/rabbit-cannon/cmd"

var version = "master"

func main() {
	cmd.Execute(version)
}
