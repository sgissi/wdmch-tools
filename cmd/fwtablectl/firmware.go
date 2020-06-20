package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	"github.com/sgissi/wdmch-tools/pkg/fwtable"
)

func firmware(args []string) {
	switch args[0] {
	case "new":
		if len(args) != 6 {
			fmt.Println("Wrong number of parameters for 'firmware new'")
			usage()
		}
		firmwareNew(args[1:])
	case "types":
		fmt.Println("Firmware Types:")
		for _, t := range fwtable.FwTypes {
			fmt.Println(" ", t)
		}
	case "update":
		if len(args) != 4 {
			fmt.Println("Wrong number of parameters for 'firmware update'")
			usage()
		}
		firmwareUpdate(args[1:])
	case "remove":
		if len(args) != 3 {
			fmt.Println("Wrong number of parameters for 'firmware remove'")
			usage()
		}
		firmwareRemove(args[1:])
	}
}

func firmwareNew(args []string) {
	fwtFile := args[0]
	fwType := args[1]
	fwFile := args[2]
	sector, err := strconv.ParseInt(args[3], 0, 32)
	if err != nil {
		fmt.Println("Invalid number for sector:", args[3], err)
		os.Exit(1)
	}
	loadAddr, err := strconv.ParseUint(args[4], 0, 32)
	if err != nil {
		fmt.Println("Invalid number for load address:", args[4], err)
		os.Exit(1)
	}

	fw := readFwTable(fwtFile)

	fwe := fw.FindFwByType(fwType)
	if fwe != nil {
		fmt.Printf("There is already one partition of type %s. Either update or remove before adding\n", fwType)
		os.Exit(1)
	}

	f, err := os.Open(fwFile)
	if err != nil {
		fmt.Printf("Error opening file '%s': %v\n", fwFile, err)
		os.Exit(1)
	}

	fwe = &fwtable.FwEntry{
		Type:          fwType,
		Lzma:          false,
		ReadOnly:      true,
		Version:       0,
		DiskOffset:    int(sector) * 512,
		TargetAddress: uint32(loadAddr),
	}

	err = fwe.ReadFile(bufio.NewReader(f))
	if err != nil {
		fmt.Printf("Error reading '%s': %v\n", fwFile, err)
		os.Exit(1)
	}
	fw.Fws = append(fw.Fws, fwe)

	writeFwTable(fwtFile, fw)
}

func firmwareUpdate(args []string) {
	fwtFile := args[0]
	fwType := args[1]
	fwFile := args[2]

	fw := readFwTable(fwtFile)
	fwe := fw.FindFwByType(fwType)
	if fwe == nil {
		fmt.Printf("Could not find an entry for type %s. Use 'firmware new' to create.\n", fwType)
		os.Exit(1)
	}

	f, err := os.Open(fwFile)
	if err != nil {
		fmt.Printf("Error opening file '%s': %v\n", fwFile, err)
		os.Exit(1)
	}
	err = fwe.ReadFile(bufio.NewReader(f))
	if err != nil {
		fmt.Printf("Error reading '%s': %v\n", fwFile, err)
		os.Exit(1)
	}
	writeFwTable(fwtFile, fw)
}

func firmwareRemove(args []string) {
	fwtFile := args[0]
	fwType := args[1]

	fw := readFwTable(fwtFile)

	idx := -1
	for i, f := range fw.Fws {
		if f.Type == fwType {
			idx = i
			break
		}
	}
	if idx == -1 {
		fmt.Printf("Could not find firmware entry with type '%s'\n", fwType)
		os.Exit(1)
	}
	if len(fw.Fws) == 1 {
		fw.Fws = nil
	} else {
		// Move items after index back one position
		copy(fw.Fws[idx:], fw.Fws[idx+1:])
		// Release last item
		fw.Fws[len(fw.Fws)-1] = nil
		// Truncate
		fw.Fws = fw.Fws[:len(fw.Fws)-1]
	}
	// Save table
	writeFwTable(fwtFile, fw)
}

func writeFwTable(file string, fw *fwtable.FwTable) {
	err := fw.WriteContent()
	if err != nil {
		fmt.Println("Error updating table:", err)
		os.Exit(1)
	}

	errs := fw.Validate()
	if errs != nil {
		fmt.Println("Error validating new table:")
		for _, e := range errs {
			fmt.Println(" ", e)
		}
		os.Exit(1)
	}

	f, err := os.OpenFile(file, os.O_RDWR, 0)
	if err != nil {
		fmt.Printf("Could not open '%s' for writing: %v\n", file, err)
		os.Exit(1)
	}
	err = fw.Export(f)
	if err != nil {
		fmt.Printf("Error writing table to '%s': %v\n", file, err)
		os.Exit(1)
	}
}

func readFwTable(file string) *fwtable.FwTable {
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		fmt.Printf("Could not open '%s' for reading: %v\n", file, err)
		os.Exit(1)
	}
	fw, err := fwtable.New(f)
	if err != nil {
		fmt.Println("Error reading FWTable:", err)
		os.Exit(1)
	}
	return fw
}
