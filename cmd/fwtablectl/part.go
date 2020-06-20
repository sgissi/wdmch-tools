package main

import (
	"fmt"
	"os"
	"strconv"
)

func part(args []string) {
	switch args[0] {
	case "remove":
		if len(args) != 3 {
			fmt.Println("Wrong number of arguments for 'parts remove'")
			usage()
		}
		partRemove(args[1:])
	}
}

func partRemove(args []string) {
	fwtFile := args[0]
	index, err := strconv.ParseInt(args[1], 10, 8)
	if err != nil {
		fmt.Println("Invalid number for index:", args[1], err)
		os.Exit(1)
	}
	idx := int(index)
	fw := readFwTable(fwtFile)
	if idx < 1 || idx > len(fw.Parts) {
		fmt.Println("Invalid part index:", idx)
		os.Exit(1)
	}
	if len(fw.Parts) == 1 {
		fw.Parts = nil
	} else {
		// Move items after index back one position
		copy(fw.Parts[idx:], fw.Parts[idx+1:])
		// Release last item
		fw.Parts[len(fw.Parts)-1] = nil
		// Truncate
		fw.Parts = fw.Parts[:len(fw.Parts)-1]
	}
	// Save table
	writeFwTable(fwtFile, fw)
}
