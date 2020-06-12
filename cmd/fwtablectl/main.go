package main

import (
	"fmt"
	"os"

	"github.com/sgissi/wdmch-tools/pkg/fwtable"
)

func usage() {
	fmt.Println("Usage: fwtablectl command")
	fmt.Println("Commands:")
	fmt.Println("  show [fwtable-file] - Read from file and show existing entries")
	fmt.Println("  firmware new    [fwtable-file] [type] [file] [sector] [load-address] - Add new firmware")
	fmt.Println("  firmware update [fwtable-file] [type] [file] ....................... - Update entry with new file")
	fmt.Println("  firmware remove [fwtable-file] [type] .............................. - Remove firmware from table")
	fmt.Println("  firmware types  .................................................... - Print available firmware types")
	fmt.Println("---")
	fmt.Println("Examples:")
	fmt.Println("  fwtablectl show /dev/sataa1")
	fmt.Println("  fwtablectl firmware update fwtable.bin KernelRootFS rootfs.cpio.gz")
	fmt.Println("  fwtablectl firmware new /dev/sataa1 KernelRootFS rootfs.cpio.gz 67584 0x02200000")
	fmt.Println("  fwtablectl firmware remove /root/fwtable.bin uBoot")
	os.Exit(1)
}

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

func main() {

	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "show":
		if len(os.Args) != 3 {
			usage()
		}
		show(os.Args[2])
	case "firmware":
		if len(os.Args) < 3 {
			usage()
		}
		firmware(os.Args[2:])
	default:
		fmt.Println("Invalid command:", os.Args[1])
		usage()
	}
}

func show(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Could not open file: %v\n", err)
		os.Exit(1)
	}

	fw, err := fwtable.New(f)
	if err != nil {
		fmt.Printf("Error reading FW table: %v\n", err)
		os.Exit(1)
	}
	print(fw)
}

func print(fw *fwtable.FwTable) {
	errs := fw.Validate()
	if errs != nil {
		fmt.Println("Validation Errors:")
		for _, e := range errs {
			fmt.Println(" -", e)
		}
	}

	fmt.Printf("Signature: %s Checksum 0x%08x Size %d Version %d Extra data: %d\n", fw.Signature, fw.Checksum, fw.Size, fw.Version, fw.Extra)
	for i, part := range fw.Parts {
		fmt.Printf("Part %d: %s (%d) RO: %t Size: %d - Mount Point: '%s' (%d) Format: %s\n", i+1, part.Type, part.FwCount, part.ReadOnly, part.Lenght, part.MountPoint, part.EmmcPartID, part.FwType)
	}
	for i, f := range fw.Fws {
		fmt.Printf("Firmware %d: %s RO:%t Compressed:%t Version: %d Size: %d (%d with padding) Disk Offset: %d (sector %d) Load Address: 0x%08x Checksum: 0x%08x\n", i+1, f.Type, f.ReadOnly, f.Lzma, f.Version, f.Length, f.Paddings, f.DiskOffset, f.DiskOffset/512, f.TargetAddress, f.Checksum)
	}
}
