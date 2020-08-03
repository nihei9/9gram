package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/nihei9/9gram/grammar"
	"github.com/nihei9/9gram/log"
	"github.com/nihei9/9gram/parser"
)

func main() {
	os.Exit(doMain())
}

func doMain() int {
	flag.Parse()

	err := run(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return 0
}

func run(args []string) error {
	var src io.Reader
	if len(args) > 0 {
		filepath := args[0]
		file, err := os.Open(filepath)
		if err != nil {
			return err
		}
		defer file.Close()
		src = file
	} else {
		src = os.Stdin
	}

	err := log.Init("9gram.log")
	if err != nil {
		return err
	}
	defer log.Close()

	psr, err := parser.NewParser(src)
	if err != nil {
		log.Log("Failed to craete a parser: %v", err)
		return err
	}
	ast, err := psr.Parse()
	if err != nil {
		log.Log("Failed to parse: %v", err)
		return err
	}

	gram, err := grammar.GenGrammar(ast)
	if err != nil {
		log.Log("Failed to generate a grammar information: %v", err)
		return err
	}

	tab, err := grammar.GenTable(gram)
	if err != nil {
		log.Log("Failed to generate a parsing table: %v", err)
		return err
	}

	d, err := grammar.GenJSON(gram, tab)
	if err != nil {
		log.Log("Failed to generate a JSON output: %v", err)
		return err
	}
	fmt.Println(string(d))

	return nil
}
